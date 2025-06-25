package handler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	contextkey "github.com/jjhwan-h/bundle-server/api/context"
	"github.com/jjhwan-h/bundle-server/config"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
	"github.com/jjhwan-h/bundle-server/domain/usecase"
	"github.com/jjhwan-h/bundle-server/internal/clients"
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"
	"github.com/jjhwan-h/bundle-server/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/flock"
	"go.uber.org/zap"
)

type ServiceHandler struct {
	CasbUsecase usecase.CasbUsecase
	Client      *clients.Client
	*zap.Logger
}

// @title service api
// @version 1.0
// @BasePath /services

// BuildDataNBundles godoc
// @Summary      Trigger policy DB update and generate OPA bundles
// @Description  Receives a trigger event to regenerate OPA's data.json, regular bundle, and delta bundle. <br> If changes are detected, notifies OPA SDK client via webhook (POST /hooks/bundle-update?type=delta).
//
// @Tags         service
// @Produce      json
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
//
// @Success      202 {object} httpResponse "Accepted - data.json and bundle were generated successfully"
// @Success      200 {object} httpResponse "OK - no changes detected in data.json"
// @Failure      500 {object} appErr.HttpError "Internal server error during bundle generation"
// @Router       /services/{service}/data/trigger [post]
//
// @Example Request:
// POST /services/casb/data/trigger
func (sh *ServiceHandler) BuildDataNBundles(c *gin.Context) {
	service := c.Param("service")
	dataPath := fmt.Sprintf("%s/%s/regular/data.json", config.Cfg.OpaDataPath, service)
	patchPath := fmt.Sprintf("%s/%s/delta/patch.json", config.Cfg.OpaDataPath, service)

	var data *usecase.Data
	var err error

	switch service {
	case "casb":
		// data 빌드
		data, err = sh.CasbUsecase.BuildDataJson(c)
		if err != nil {
			sh.Error("failed to build data.json", zap.Error(err))
			c.Set(contextkey.LogLevel, zap.ErrorLevel)
			c.Error(appErr.NewHttpError(
				"internal_server_error",
				http.StatusInternalServerError,
				"failed to build data.json",
			))
			return
		}

	case "ztna":
		c.Set(contextkey.LogLevel, zap.WarnLevel)
		c.Error(appErr.NewHttpError(
			"unsupported_service",
			http.StatusNotImplemented,
			"ztna service is not supported yet",
		))
		return

	default:
		sh.Warn("Invalid service parameter", zap.String("service", service))
		c.Set(contextkey.LogLevel, zap.WarnLevel)
		c.Error(appErr.NewHttpError(
			"bad_request",
			http.StatusBadRequest,
			"Invalid service parameter",
		))
		return
	}

	// delta-bundle 생성
	err = buildDeltaBundle(
		c,
		data,
		dataPath,
		patchPath,
		fmt.Sprintf("%s/%s/delta.tar.gz", config.Cfg.OpaDataPath, service),
	)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			if errors.Is(err, appErr.ErrNoChanges) { // data.json 변경 x
				sh.Info("patch.json not generated: no changes detected", zap.String("patch", patchPath))
				c.Set(contextkey.LogLevel, zap.InfoLevel)
				c.JSON(http.StatusOK,
					&httpResponse{
						Code:    "no_changes",
						Message: err.Error(),
						Status:  http.StatusOK,
					})
				return
			} else {
				sh.Error("failed to build delta bundle", zap.Error(err))
				c.Set(contextkey.LogLevel, zap.ErrorLevel)
				c.Error(appErr.NewHttpError(
					"internal_server_error",
					http.StatusInternalServerError,
					err.Error(),
				))
				return
			}
		} else {
			sh.Info("No existing data.json found. Skipping delta bundle generation", zap.String("data", dataPath))
		}
	} else {
		sh.Info("Delta Bundle created successfully", zap.String("service", service))
	}

	// 일반-bundle 생성
	// opa-sdk-client들 초기 실행 시 변경사항이 반영된 일반-bundle 필요
	err = buildBundle(
		c,
		data,
		dataPath,
		fmt.Sprintf("%s/%s/regular.tar.gz", config.Cfg.OpaDataPath, service),
	)
	if err != nil {
		sh.Error("failed to build regular bundle", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	} else {
		sh.Info("Regular Bundle created successfully", zap.String("service", service))
	}

	go func() {
		err := sh.Client.Hook("/hooks/bundle-update?type=delta", service)
		if err != nil {
			sh.Error("failed to event notification", zap.Error(err))
		}
	}()

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "success",
		Message: "data.json and bundle were generated successfully. Notification will be sent to the OPA client.",
		Status:  http.StatusAccepted,
	})
}

// CreateBundle godoc
// @Summary      Trigger policy.rego update and generate OPA bundles
// @Description  Receives a trigger event to regenerate regular bundle. <br> If changes are detected, notifies OPA SDK client via webhook (POST /hooks/bundle-update).
//
// @Tags         service
// @Produce      json
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
//
// @Success      202 {object} httpResponse "Accepted - bundle were generated successfully"
// @Failure      500 {object} appErr.HttpError "Internal server error during bundle generation"
// @Router       /services/{service}/policy/trigger [post]
//
// @Example Request:
// POST /services/casb/policy/trigger
func (sh *ServiceHandler) CreateBundle(c *gin.Context) {
	service := c.Param("service")
	err := createBundle(
		c.Request.Context(),
		fmt.Sprintf("%s/%s/regular.tar.gz", config.Cfg.OpaDataPath, service),
		fmt.Sprintf("%s/%s/regular", config.Cfg.OpaDataPath, service),
	)

	if err != nil {
		sh.Error("failed to build regular bundle", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	} else {
		sh.Info("Regular Bundle created successfully", zap.String("service", service))
	}

	go func() {
		err := sh.Client.Hook("/hooks/bundle-update", service)
		if err != nil {
			sh.Error("failed to event notification", zap.Error(err))
		}
	}()

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "success",
		Message: " bundle were generated successfully. Notification will be sent to the OPA client.",
		Status:  http.StatusAccepted,
	})
}

// RegisterPolicy godoc
// @Summary      Register policy.rego
// @Description  Uploads a policy.rego file and saves it to the service-specific bundle directory.
// @Tags         service
// @Accept       multipart/form-data
// @Produce      json
//
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
// @Param        file formData file true "The policy.rego file to upload"
//
// @Success      201 {object} httpResponse "Created - The policy file was saved successfully"
// @Failure      500 {object} appErr.HttpError "Internal server error during file saving"
//
// @Router       /services/{service}/policy [post]
//
// @Example Request:
// POST /services/casb/policy
// Content-Type: multipart/form-data
// Form field: file = policy.rego
func (sh *ServiceHandler) RegisterPolicy(c *gin.Context) {
	service := c.Param("service")
	policyPath := fmt.Sprintf("%s/%s/regular/policy.rego", config.Cfg.OpaDataPath, service)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		sh.Error("failed to parse form file", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		sh.Error("failed to open form file", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}
	defer file.Close()

	err = utils.SaveToFile(c.Request.Context(), file, policyPath)
	if err != nil {
		sh.Error("failed to save form file", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	} else {
		sh.Info("policy.rego created successfully", zap.String("service", service))
	}

	c.JSON(http.StatusCreated, &httpResponse{
		Code:    "success",
		Message: "The policy file was generated successfully.",
		Status:  http.StatusCreated,
	})
}

// ServeBundle godoc
// @Summary      Download OPA bundle file
// @Description  Serves either a regular or delta bundle file (.tar.gz) for a specific service. Use query parameter `type=delta` to get the delta bundle.
// @Tags         service
// @Produce      application/gzip
//
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
// @Param        type query string false "Bundle type: 'regular' (default) or 'delta'"
//
// @Success      200 {file} file "Bundle file (application/gzip)"
// @Failure      500 {object} appErr.HttpError "Internal server error or file not found"
//
// @Router       /services/{service}/bundle [get]
//
// @Example Request:
// GET /services/casb/bundle
// GET /services/casb/bundle?type=delta
func (sh *ServiceHandler) ServeBundle(c *gin.Context) {
	var path string
	var filename string

	service := c.Param("service")
	t := c.Query("type")

	switch t {
	case "delta":
		path = fmt.Sprintf("%s/%s/delta.tar.gz", config.Cfg.OpaDataPath, service)
		filename = fmt.Sprintf("%s_delta.tar.gz", service)
	case "", "regular":
		path = fmt.Sprintf("%s/%s/regular.tar.gz", config.Cfg.OpaDataPath, service)
		filename = fmt.Sprintf("%s_regular.tar.gz", service)
	default:
		sh.Info("Invalid bundle type, defaulting to regular bundle",
			zap.String("requested_type", t),
			zap.String("service", service),
		)
		path = fmt.Sprintf("%s/%s/regular.tar.gz", config.Cfg.OpaDataPath, service)
		filename = fmt.Sprintf("%s_regular.tar.gz", service)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		sh.Error("bundle file not found", zap.Error(err))
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			err.Error(),
		))
		return
	}

	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s", filename))
	http.ServeFile(c.Writer, c.Request, path)
}

// RegisterClients godoc
// @Summary      Register OPA webhook client addresses
// @Description  Registers one or more OPA SDK client addresses for the specified service. Accepts a JSON array of client URLs or IPs.
// @Tags         service
// @Accept       json
// @Produce      json
//
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
// @Param        clients body []string true "List of OPA client addresses (IP or domain)"
//
// @Success      200 {object} httpResponse "Clients registered successfully"
// @Failure      400 {object} appErr.HttpError "Invalid JSON format"
// @Failure      404 {object} appErr.HttpError "No clients found in request"
// @Failure      409 {object} appErr.HttpError "Conflict - client already exists or internal error"
//
// @Router       /services/{service}/clients [post]
//
// @Example Request:
// POST /services/casb/clients
// Body:
// [
//
//	"http://127.0.0.1:5556",
//	"http://opa-client.k8s.local:8181"
//
// ]
func (sh *ServiceHandler) RegisterClients(c *gin.Context) {
	service := c.Param("service")

	var clients []string

	err := c.ShouldBindJSON(&clients)
	if err != nil {
		sh.Error("Invalid reqeust body. Please check the JSON format")
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"bad_request",
			http.StatusBadRequest,
			err.Error(),
		))
		return
	}

	if len(clients) == 0 {
		msg := "no clients found"
		sh.Error(msg)
		c.Set(contextkey.LogLevel, zap.ErrorLevel)
		c.Error(appErr.NewHttpError(
			"no_data",
			http.StatusNotFound,
			msg,
		))
		return
	}

	if err = sh.Client.AddHookClient(clients, service); err != nil {
		c.Error(appErr.NewHttpError(
			"confilct",
			http.StatusConflict,
			err.Error(),
		))
		return
	}

	sh.Info("client address has been successfully registerd", zap.String("service", service))

	c.JSON(http.StatusOK, httpResponse{
		Code:    "client_registered",
		Message: "Client address has been successfully registered.",
		Status:  http.StatusOK,
	})
}

// ServeClients godoc
// @Summary      Get all registered OPA clients
// @Description  Returns a map of all registered OPA clients grouped by service.
// @Tags         service
// @Produce      json
//
// @Success      200 {object} clientGroupResponse "Map of service name to client list"
//
// @Router       /services/clients [get]
//
// @Example Request:
// GET /services/clients
func (sh *ServiceHandler) ServeClients(c *gin.Context) {
	clients := sh.Client.GetAll()
	c.JSON(http.StatusOK, clients)
}

// ServeServiceClients godoc
// @Summary      Get clients by service
// @Description  Returns a list of registered OPA client addresses for the specified service.
// @Tags         service
// @Produce      json
//
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
// @Success      200 {array} clientGroup "List of registered clients"
//
// @Router       /services/{service}/clients [get]
//
// @Example Request:
// GET /services/casb/clients
func (sh *ServiceHandler) ServeServiceClients(c *gin.Context) {
	service := c.Param("service")

	clients := sh.Client.Get(service)
	c.JSON(http.StatusOK, clients)
}

// DeleteClients godoc
// @Summary      Delete one or all OPA clients for a service
// @Description  Deletes a specific client (by IP or DNS) or all clients for a service if no client is specified.
// @Tags         service
// @Produce      json
//
// @Param        service path string true "Service name <br> Only services listed in clients.service of the config file are allowed."
// @Param        client query string false "Client address (IP or domain). If omitted, all clients will be deleted."
//
// @Success      200 {object} httpResponse "Client(s) deleted successfully"
// @Failure      404 {object} httpResponse "Client not found"
//
// @Router       /services/{service}/clients [delete]
//
// @Example Request:
// DELETE /services/casb/clients?client=http://127.0.0.1:5556
// DELETE /services/casb/clients
func (sh *ServiceHandler) DeleteClients(c *gin.Context) {
	t := c.Query("client")
	service := c.Param("service")

	if t != "" {
		err := sh.Client.Delete(service, t)
		if err != nil {
			sh.Error("client not found", zap.String("service", service), zap.String("ip", t))
			c.Set(contextkey.LogLevel, zap.ErrorLevel)
			c.Error(appErr.NewHttpError(
				"client_not_found",
				http.StatusNotFound,
				"failed to delete client.",
			))
			return
		}
		sh.Info("client deleted successfully", zap.String("servcie", service), zap.String("ip", t))
		c.JSON(http.StatusOK, httpResponse{
			Code:    "delete_successfully",
			Message: "client deleted successfully",
			Status:  http.StatusOK,
		})
		return
	} else { // 전체삭제
		sh.Client.DeleteAll(service)
		sh.Info("all clients deleted successfully", zap.String("service", service))
		c.JSON(http.StatusOK, httpResponse{
			Code:    "delete_successfully",
			Message: "all clients deleted successfully",
			Status:  http.StatusOK,
		})
		return
	}
}

func buildDeltaBundle(ctx context.Context, data *usecase.Data, dataPath, patchPath, tarGzPath string) error {
	byteOldData, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("%s: %w", "failed to read data.json", err)
	}

	var oldData usecase.Data
	if err := json.Unmarshal(byteOldData, &oldData); err != nil {
		return fmt.Errorf("%s: %w", "failed to unmarshal data to map", err)
	}

	//patch.json 생성
	patch, err := buildPatchJson(&oldData, data)
	if err != nil {
		return fmt.Errorf("patch.json not generated: %w", err)
	}
	buf := new(bytes.Buffer)
	err = utils.EncodeJson(buf, patch)
	if err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrEncodeData.Error(), err)
	}

	if err := utils.SaveToFile(ctx, buf, patchPath); err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrSaveData.Error(), err)
	}

	//delta-bundle 생성
	err = createBundle(
		ctx,
		tarGzPath,
		filepath.Dir(patchPath),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrBuildBundle.Error(), err)
	}

	return nil
}

func buildBundle(ctx context.Context, data *usecase.Data, dataPath, tarGzPath string) error {
	//json형식으로 인코딩
	buf := new(bytes.Buffer)
	err := utils.EncodeJson(buf, data)
	if err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrEncodeData.Error(), err)
	}

	//data.json 저장
	if err := utils.SaveToFile(ctx, buf, dataPath); err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrSaveData.Error(), err)
	}

	//일반-bundle 생성
	err = createBundle(
		ctx,
		tarGzPath,
		filepath.Dir(dataPath),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrBuildBundle.Error(), err)
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
	if len(files) == 0 {
		return fmt.Errorf("source directory has no file")
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
		if file.IsDir() || strings.HasSuffix(file.Name(), ".lock") {
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
