package handler

import (
	"archive/tar"
	"bundle-server/domain/integration/category"
	"bundle-server/domain/usecase"
	appErr "bundle-server/internal/errors"
	"bundle-server/internal/utils"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type DataHandler struct {
	CasbUsecase usecase.CasbUsecase
}

func (dh *DataHandler) BuildDataJson(c *gin.Context) {
	dataPath := fmt.Sprintf("%s/sse/regular/data.json", viper.GetString("OPA_DATA_PATH"))
	patchPath := fmt.Sprintf("%s/sse/delta/patch.json", viper.GetString("OPA_DATA_PATH"))

	//data 빌드
	data, err := dh.CasbUsecase.BuildDataJson(c)
	if err != nil {
		log.Printf("[ERROR] %s: %v\n", appErr.ErrBuildData.Error(), err)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}

	// delta-bundle 생성 및 알림
	go func(data *usecase.Data, dataPath, patchPath string) {
		err := buildDeltaBundle(data, dataPath, patchPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Println("initial build: data.json missing, proceeding to generate", err)
			} else {
				log.Println(err)
				return
			}
		}

		//opa-sdk-client에 알림

	}(data, dataPath, patchPath)

	// 일반-bundle 생성
	// opa-sdk-client들 초기 실행 시 변경사항이 반영된 일반-bundle 필요
	go func(data *usecase.Data, dataPath string) {
		err := buildBundle(data, dataPath)
		if err != nil {
			log.Println(err)
			return
		}
	}(data, dataPath)

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "BUILD_DATA_JSON",
		Message: "data.json is being saved in the background",
		Status:  http.StatusAccepted,
	})
}

func buildDeltaBundle(data *usecase.Data, dataPath, patchPath string) error {
	byteOldData, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", "failed to read data.json", err)
	}

	var oldData usecase.Data
	if err := json.Unmarshal(byteOldData, &oldData); err != nil {
		return fmt.Errorf("[ERROR] %s: %w", "failed to unmarshal data to map", err)
	}

	//patch.json 생성
	patch, err := buildPatchJson(&oldData, data)
	if err != nil {
		return fmt.Errorf("[ERROR] failed to build patch.json: %w", err)
	}
	buf := new(bytes.Buffer)
	err = utils.EncodeJson(buf, patch)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrEncodeData.Error(), err)
	}

	if err := utils.SaveToFile(buf, patchPath); err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrSaveData.Error(), err)
	}

	//delta-bundle 생성
	err = createBundle(
		fmt.Sprintf("%s/sse/bundle/%s", viper.GetString("OPA_DATA_PATH"), "casb_delta.tar.gz"),
		fmt.Sprintf("%s/sse/delta", viper.GetString("OPA_DATA_PATH")),
	)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrBuildBundle.Error(), err)
	}

	return nil
}

func buildBundle(data *usecase.Data, dataPath string) error {
	//json형식으로 인코딩
	buf := new(bytes.Buffer)
	err := utils.EncodeJson(buf, data)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrEncodeData.Error(), err)
	}

	//data.json 저장
	if err := utils.SaveToFile(buf, dataPath); err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrSaveData.Error(), err)
	}

	//일반-bundle 생성
	err = createBundle(
		fmt.Sprintf("%s/sse/bundle/%s", viper.GetString("OPA_DATA_PATH"), "casb_regular.tar.gz"),
		fmt.Sprintf("%s/sse/regular", viper.GetString("OPA_DATA_PATH")),
	)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrBuildBundle.Error(), err)
	}

	return nil
}

func createBundle(tarGzPath, sourceDir string) error {
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(tarGzPath), 0755)
	if err != nil {
		return err
	}

	tarFile, err := os.Create(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz: %w", err)
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, file := range files {
		if file.IsDir() {
			continue // 디렉토리는 무시 (필요 없음)
		}
		filePath := filepath.Join(sourceDir, file.Name())

		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, file.Name())
		if err != nil {
			return err
		}

		header.Name = file.Name()
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}
	}

	return nil
}

func buildPatchJson(oldData *usecase.Data, data *usecase.Data) (*patch, error) {
	patchData := getChanges(oldData, data)
	if len(patchData) == 0 {
		return nil, fmt.Errorf("no changes to generate patch")
	}
	return &patch{
		Data: patchData,
	}, nil
}

func getChanges(oldData *usecase.Data, data *usecase.Data) (changes []patchData) {
	// default_effect
	if oldData.DefaultEffect != data.DefaultEffect {
		changes = append(changes, patchData{
			Op:    "replace",
			Path:  "/default_effect",
			Value: data.DefaultEffect,
		})
	}
	// policies
	for _, newPolicy := range data.Policies {
		isExist := false
		for idx, oldPolicy := range oldData.Policies {
			if newPolicy.PolicyID == oldPolicy.PolicyID {
				isExist = true
				changes = append(changes, comparePolicies(oldPolicy, newPolicy, idx)...)
				break
			}
		}
		if !isExist {
			changes = append(changes, patchData{"upsert", "/policies", newPolicy})
		}
	}

	for idx, oldPolicy := range oldData.Policies {
		isExist := false
		for _, newPolicy := range data.Policies {
			if oldPolicy.PolicyID == newPolicy.PolicyID {
				isExist = true
				break
			}
		}
		if !isExist {
			changes = append(changes, patchData{"remove", fmt.Sprintf("/policies/%d", idx), nil})
		}
	}
	return
}

func comparePolicies(oldPolicy, newPolicy usecase.Policy, idx int) (changes []patchData) {
	prefix := fmt.Sprintf("/policies/%d", idx)

	if newPolicy.PolicyID != oldPolicy.PolicyID {
		changes = append(changes, patchData{"replace", prefix + "/id", newPolicy.PolicyID})
	}
	if newPolicy.Priority != oldPolicy.Priority {
		changes = append(changes, patchData{"replace", prefix + "/priority", newPolicy.Priority})
	}
	if newPolicy.PolicyName != oldPolicy.PolicyName {
		changes = append(changes, patchData{"replace", prefix + "/name", newPolicy.PolicyName})
	}
	if newPolicy.Effect != oldPolicy.Effect {
		changes = append(changes, patchData{"replace", prefix + "/effect", newPolicy.Effect})
	}
	if !slices.Equal(newPolicy.Subject.Users, oldPolicy.Subject.Users) {
		changes = append(changes, patchData{"replace", prefix + "/subject/users", newPolicy.Subject.Users})
	}
	if !slices.Equal(newPolicy.Subject.Groups, oldPolicy.Subject.Groups) {
		changes = append(changes, patchData{"replace", prefix + "/subject/groups", newPolicy.Subject.Groups})
	}

	if !equalService(newPolicy.Services, oldPolicy.Services) {
		changes = append(changes, patchData{"replace", prefix + "/services", newPolicy.Services})
	}

	return
}

// 내부 객체 값은 동일하지만 객체 순서가 바뀐 경우에도 다른 것으로 처리됨.
func equalService(a, b []category.CategoryService) bool {
	return reflect.DeepEqual(a, b)
}
