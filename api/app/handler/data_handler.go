package handler

import (
	"archive/tar"
	"bundle-server/domain/integration/category"
	"bundle-server/domain/usecase"
	appErr "bundle-server/internal/errors"
	"bundle-server/internal/utils"
	"bytes"
	"compress/gzip"
	"context"
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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/flock"
	"github.com/spf13/viper"
)

type DataHandler struct {
	CasbUsecase usecase.CasbUsecase
}

func (dh *DataHandler) BuildDataJson(c *gin.Context) {
	/* TODO: sse 서비스마다 path 또는 query로 서비스 명을 받아 저장 경로를 다르게 지정
	예를들어, casb
	 /var/lib/opa/sse/casb/data.json
	 /var/lib/opa/sse/casb/patch.json
	 /var/lib/opa/sse/casb/regular_bundle.tar.gz
	 /var/lib/opa/sse/casb/delta_bundle.tar.gz

	 ztna
	 /var/lib/opa/sse/ztna/data.json
	 /var/lib/opa/sse/ztna/patch.json
	 /var/lib/opa/sse/ztna/regular_bundle.tar.gz
	 /var/lib/opa/sse/ztna/delta_bundle.tar.gz

	 ...
	*/
	dataPath := fmt.Sprintf("%s/data/%s_data.json", viper.GetString("OPA_DATA_PATH"), "casb")
	patchPath := fmt.Sprintf("%s/data/%s_patch.json", viper.GetString("OPA_DATA_PATH"), "casb")

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

	// delta-bundle 생성
	err = buildDeltaBundle(c, data, dataPath, patchPath)
	if err != nil {
		log.Println("delta bundle: ", err)
		if !errors.Is(err, os.ErrNotExist) {
			if errors.Is(err, appErr.ErrNoChanges) {
				c.Error(appErr.NewHttpError(
					"no_changes",
					http.StatusOK,
					err.Error(),
				))
				return
			} else {
				c.Error(appErr.NewHttpError(
					"internal_server_error",
					http.StatusInternalServerError,
					err.Error(),
				))
				return
			}
		}
	}

	// 일반-bundle 생성
	// opa-sdk-client들 초기 실행 시 변경사항이 반영된 일반-bundle 필요
	err = buildBundle(c, data, dataPath)
	if err != nil {
		log.Println("regular bundle: ", err)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}

	// go func() {
	// 	//opa-sdk-client에 알림
	// }()

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "success",
		Message: "data.json and bundle were generated successfully.",
		Status:  http.StatusAccepted,
	})
}

func buildDeltaBundle(ctx context.Context, data *usecase.Data, dataPath, patchPath string) error {
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
		return fmt.Errorf("patch.json not generated: %w", err)
	}
	buf := new(bytes.Buffer)
	err = utils.EncodeJson(buf, patch)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrEncodeData.Error(), err)
	}

	if err := utils.SaveToFile(ctx, buf, patchPath); err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrSaveData.Error(), err)
	}

	//delta-bundle 생성
	err = createBundle(
		ctx,
		fmt.Sprintf("%s/%s_delta.tar.gz", filepath.Dir(patchPath), "casb"), // TODO: 변경필요
		filepath.Dir(patchPath),
	)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrBuildBundle.Error(), err)
	}

	return nil
}

func buildBundle(ctx context.Context, data *usecase.Data, dataPath string) error {
	//json형식으로 인코딩
	buf := new(bytes.Buffer)
	err := utils.EncodeJson(buf, data)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrEncodeData.Error(), err)
	}

	//data.json 저장
	if err := utils.SaveToFile(ctx, buf, dataPath); err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrSaveData.Error(), err)
	}

	//일반-bundle 생성
	err = createBundle(
		ctx,
		fmt.Sprintf("%s/%s_regular.tar.gz", filepath.Dir(dataPath), "casb"), // TODO: 변경필요
		filepath.Dir(dataPath),
	)
	if err != nil {
		return fmt.Errorf("[ERROR] %s: %w", appErr.ErrBuildBundle.Error(), err)
	}

	return nil
}

func createBundle(ctx context.Context, tarGzPath, sourceDir string) error {
	err := os.MkdirAll(filepath.Dir(tarGzPath), 0755)
	if err != nil {
		return err
	}

	lock := flock.New(tarGzPath + ".lock")
	locked, err := lock.TryLockContext(ctx, time.Millisecond*500)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("resource busy: tar file is locked")
	}
	defer lock.Unlock()

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	tmpPath := tarGzPath + ".tmp"
	tarFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create tmp tar.gz: %w", err)
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, file := range files {
		if file.IsDir() {
			continue
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
		_, err = io.Copy(tarWriter, f)
		f.Close()
		if err != nil {
			return err
		}
	}

	return os.Rename(tmpPath, tarGzPath)
}

func buildPatchJson(oldData *usecase.Data, data *usecase.Data) (*patch, error) {
	patchData := getChanges(oldData, data)
	if len(patchData) == 0 {
		return nil, appErr.ErrNoChanges
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
