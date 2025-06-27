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
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	contextkey "github.com/jjhwan-h/bundle-server/api/context"
	"github.com/jjhwan-h/bundle-server/config"
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
// @Description  Triggers OPA bundle regeneration.
// @Description  1. Builds `data.json` for the given service
// @Description  2. Compares with previous version to generate `patch.json`
// @Description  3. If changes are found, creates `delta.tar.gz` and `regular-vX.X.tar.gz` bundles
// @Description  4. Sends webhook POST /hooks/bundle-update?type=delta to notify OPA SDK clients
//
// @Tags         service
// @Accept       json
// @Produce      json
// @Param        service path string true "Service name (only services defined in config.clients.service are allowed)"
//
// @Success      202 {object} httpResponse "Accepted - Bundles generated and notification will be sent to OPA clients"
// @Success      200 {object} httpResponse "OK - No changes detected in data.json (no new bundles created)"
// @Failure      400 {object} appErr.HttpError "Invalid service parameter"
// @Failure      500 {object} appErr.HttpError "Internal server error during data/bundle generation"
// @Failure      501 {object} appErr.HttpError "Service not yet supported"
//
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
			appErr.HandleError(c, sh.Logger, appErr.HttpError{
				Code:   "internal_server_error",
				Status: http.StatusInternalServerError,
				Err:    "failed to build data.json",
			}, "failed to build data.json", zap.String("service", service))
			return
		}

	case "ztna":
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "unsupported_service",
			Status: http.StatusNotImplemented,
			Err:    "ztna service is not supported yet",
		}, "ztna service is not supported yet", zap.String("service", service))
		return

	default:
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "bad_request",
			Status: http.StatusBadRequest,
			Err:    "Invalid service parameter",
		}, "Invalid service parameter", zap.String("service", service))
		return
	}

	oldData, err := getDataJson(dataPath)
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to Read Data.json", zap.Error(err), zap.String("service", service))
		return
	}

	//patch.json 생성
	patch, err := sh.CasbUsecase.BuildPatchJson(oldData, data)
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to build patch.json", zap.Error(err), zap.String("service", service))
		return
	}
	// delta-bundle 생성
	err = buildDeltaBundle(
		c,
		patch,
		patchPath,
		fmt.Sprintf("%s/%s/delta.tar.gz", config.Cfg.OpaDataPath, service),
	)
	if err != nil {
		// data.json 없음: 로깅만 하고 아래로 진행
		if errors.Is(err, os.ErrNotExist) {
			sh.Info("No existing data.json found. Skipping delta bundle generation", zap.String("data", dataPath))
		} else if errors.Is(err, appErr.ErrNoChanges) {
			// 변경 없음: return with 200
			sh.Info("patch.json not generated: no changes detected", zap.String("patch", patchPath))
			c.Set(contextkey.LogLevel, zap.InfoLevel)
			c.JSON(http.StatusOK, &httpResponse{
				Code:    "no_changes",
				Message: err.Error(),
				Status:  http.StatusOK,
			})
			return
		} else {
			appErr.HandleError(c, sh.Logger, appErr.HttpError{
				Code:   "internal_server_error",
				Status: http.StatusInternalServerError,
				Err:    err.Error(),
			}, "failed to build delta-bundle", zap.Error(err), zap.String("service", service))
			return
		}
	}
	sh.Info("Delta Bundle created successfully", zap.String("service", service))

	nMajor, nMinor := sh.Client.Bundle[service].Latest.NextVersion()
	// 일반-bundle 생성
	// opa-sdk-client들 초기 실행 시 변경사항이 반영된 일반-bundle 필요
	err = buildBundle(
		c,
		data,
		dataPath,
		fmt.Sprintf("%s/%s/regular-v%d.%d.tar.gz", config.Cfg.OpaDataPath, service, nMajor, nMinor),
	)
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to build regular bundle", zap.Error(err), zap.String("service", service))
		return
	} else {
		sh.Info("Regular Bundle created successfully", zap.String("service", service))
		sh.Client.Bundle[service].Latest.IncrementVersion()
	}

	// Etag update
	_, err = sh.Client.Bundle[service].ETagFromFile()
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to update etag(hash)", zap.Error(err), zap.String("service", service))
		return
	}

	go func(major int, minor int8) {
		err := sh.Client.Hook("hooks/bundle-update?type=delta", service)
		if err != nil {
			sh.Error("failed to event notification", zap.Error(err))
		}
	}(nMajor, nMinor)

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "success",
		Message: "data.json and bundle were generated successfully. Notification will be sent to the OPA client.",
		Status:  http.StatusAccepted,
	})
}

// CreateBundle godoc
// @Summary      Trigger policy.rego update and generate regular OPA bundle
// @Description  Triggers regeneration of the regular bundle (policy.rego and related files).
// @Description  If changes are detected, sends a webhook notification to OPA SDK clients via POST /hooks/bundle-update.
//
// @Tags         service
// @Accept       json
// @Produce      json
// @Param        service path string true "Service name (only services defined in config.clients.service are allowed)"
//
// @Success      202 {object} httpResponse "Accepted - Regular bundle was generated and notification will be sent to OPA clients"
// @Failure      400 {object} appErr.HttpError "Invalid service parameter"
// @Failure      500 {object} appErr.HttpError "Internal server error during regular bundle generation"
//
// @Router       /services/{service}/policy/trigger [post]
//
// @Example Request:
// POST /services/casb/policy/trigger
func (sh *ServiceHandler) CreateBundle(c *gin.Context) {
	service := c.Param("service")

	// IncrementVersion() 호출 전까지 race-condition발생 가능하므로 regular-bundle로 .lock파일 유지
	nMajor, nMinor := sh.Client.Bundle[service].Latest.NextVersion()

	err := createBundle(
		c.Request.Context(),
		fmt.Sprintf("%s/%s/regular-v%d.%d.tar.gz", config.Cfg.OpaDataPath, service, nMajor, nMinor),
		fmt.Sprintf("%s/%s", config.Cfg.OpaDataPath, service),
	)

	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to build regular bundle", zap.Error(err), zap.String("service", service))
		return
	} else {
		sh.Info("Regular Bundle created successfully", zap.String("service", service))
		sh.Client.Bundle[service].Latest.IncrementVersion()
	}

	// Etag update
	_, err = sh.Client.Bundle[service].ETagFromFile()
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to update etag(hash)", zap.Error(err), zap.String("service", service))
		return
	}

	go func(major int, minor int8) {
		err := sh.Client.Hook("hooks/bundle-update", service)
		if err != nil {
			sh.Error("failed to event notification", zap.Error(err))
		}
	}(nMajor, nMinor)

	c.JSON(http.StatusAccepted, &httpResponse{
		Code:    "success",
		Message: " bundle were generated successfully. Notification will be sent to the OPA client.",
		Status:  http.StatusAccepted,
	})
}

// RegisterPolicy godoc
// @Summary      Upload a policy.rego file
// @Description  Uploads a `policy.rego` file via multipart/form-data and saves it into the service-specific bundle directory.
// @Description  Only services defined in `clients.service` of the config file are allowed.
//
// @Tags         service
// @Accept       multipart/form-data
// @Produce      json
//
// @Param        service path string true "Service name (must be listed in config.clients.service)"
// @Param        file formData file true "policy.rego file to upload"
//
// @Success      201 {object} httpResponse "Created - The policy.rego file was saved successfully"
// @Failure      400 {object} appErr.HttpError "Invalid service parameter or malformed request"
// @Failure      500 {object} appErr.HttpError "Internal server error while saving the policy file"
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
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to parse form file", zap.Error(err), zap.String("service", service))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to open form file", zap.Error(err), zap.String("service", service))
		return
	}
	defer file.Close()

	err = utils.SaveToFileWithLock(c.Request.Context(), file, policyPath)
	if err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "internal_server_error",
			Status: http.StatusInternalServerError,
			Err:    err.Error(),
		}, "failed to save form file", zap.Error(err), zap.String("service", service))
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
// @Description  Downloads an OPA bundle file (.tar.gz) for the specified service.
// @Description  By default, the latest **regular** bundle is served.
// @Description  To download a **delta** bundle, use the query `?type=delta`.
// @Description  To request a specific version of the regular bundle, use the query `?version=X.Y`.
// @Description  Supports ETag validation using the `If-None-Match` header.
//
// @Tags         service
// @Produce      application/gzip
//
// @Param        service path string true "Service name (must be listed in config.clients.service)"
// @Param        type query string false "Bundle type: 'regular' (default) or 'delta'"
// @Param        version query string false "Regular bundle version in format 'X.Y' (e.g., 1.2)"
//
// @Success      200 {file} file "The requested bundle file (.tar.gz)"
// @Success      304 {object} httpResponse "Not Modified - Client already has the latest bundle"
// @Failure      400 {object} appErr.HttpError "Invalid service parameters or file not found"
// @Failure      500 {object} appErr.HttpError "Internal server error while serving the bundle"
//
// @Header       200 {string} ETag "ETag header containing current bundle hash"
//
// @Router       /services/{service}/bundle [get]
//
// @Example Request:
// GET /services/casb/bundle
// GET /services/casb/bundle?type=delta
// GET /services/casb/bundle?version=1.3
func (sh *ServiceHandler) ServeBundle(c *gin.Context) {
	var path string
	var filename string
	var etag string

	service := c.Param("service")
	version := c.Query("version")
	t := c.Query("type")

	major := sh.Client.Bundle[service].Latest.GetMajor()
	minor := sh.Client.Bundle[service].Latest.GetMinor()

	switch t {
	case "delta":
		path = fmt.Sprintf("%s/%s/delta.tar.gz", config.Cfg.OpaDataPath, service)
		filename = fmt.Sprintf("%s_delta.tar.gz", service)
	case "", "regular": // type이 비어있거나 regular인 경우 => regular-bundle 리턴

		// If-Non-Match 헤더와 비교
		etag = sh.Client.Bundle[service].GetEtag()
		clientEtag := c.GetHeader("If-None-Match")

		sh.Debug("etag", zap.String("etag", etag), zap.String("clientEtag", clientEtag))
		if etag == clientEtag {
			c.Set(contextkey.LogLevel, zap.InfoLevel)
			c.JSON(http.StatusNotModified, &httpResponse{
				Code:    "not_modified",
				Message: "To request the latest bundle, please omit the version query parameter.",
				Status:  http.StatusNotModified,
			})
			return
		}

		s := strings.Split(version, ".")

		// version이 비어있는경우 또는 latest 를 요청하는 경우
		if version == "" ||
			(s[0] == strconv.Itoa(major) && s[1] == strconv.Itoa(int(minor))) {
			path = fmt.Sprintf("%s/%s/regular-v%d.%d.tar.gz", config.Cfg.OpaDataPath, service, major, minor)
			filename = fmt.Sprintf("%s_regular-v%d.%d.tar.gz", service, major, minor)
			c.Header("ETag", etag) // 최신번들 요청일 경우에만 삽입
		} else {
			path = fmt.Sprintf("%s/%s/regular-v%s.%s.tar.gz", config.Cfg.OpaDataPath, service, s[0], s[1])
			filename = fmt.Sprintf("%s_regular-v%s.%s.tar.gz", service, s[0], s[1])
		}
	default:
		sh.Info("Invalid bundle type, defaulting to regular bundle",
			zap.String("requested_type", t),
			zap.String("service", service),
		)
		path = fmt.Sprintf("%s/%s/regular-v%d.%d.tar.gz", config.Cfg.OpaDataPath, service, major, minor)
		filename = fmt.Sprintf("%s_regular-v%d.%d.tar.gz", service, major, minor)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "bad_request",
			Status: http.StatusBadRequest,
			Err:    err.Error(),
		}, "bundle file not found", zap.Error(err), zap.String("service", service))
		return
	}

	sh.Info("serve bundle", zap.String("name", filename))

	mime := mime.TypeByExtension(filepath.Ext(filename))
	c.Header("Content-Type", mime)
	c.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s", filename))
	http.ServeFile(c.Writer, c.Request, path)
}

// RegisterClients godoc
// @Summary      Register OPA webhook client addresses
// @Description  Registers one or more OPA SDK client addresses for the specified service.
// @Description  Accepts a JSON array of client URLs (e.g., IP or domain).
// @Description  These clients will be notified via webhook when a new bundle is available.
//
// @Tags         service
// @Accept       json
// @Produce      json
//
// @Param        service path string true "Service name (must be listed in config.clients.service)"
// @Param        clients body []string true "List of OPA client addresses (IP or domain)"
//
// @Success      200 {object} httpResponse "Clients registered successfully"
// @Failure      400 {object} appErr.HttpError "Invalid service parameters, JSON format or no clients provided"
// @Failure      409 {object} appErr.HttpError "Conflict - one or more clients already registered or registration failed"
//
// @Router       /services/{service}/clients [post]
//
// @Example Request:
// POST /services/casb/clients
// Content-Type: application/json
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
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "bad_request",
			Status: http.StatusBadRequest,
			Err:    err.Error(),
		}, "Invalid reqeust body. Please check the JSON format", zap.Error(err), zap.String("service", service))
		return
	}

	if len(clients) == 0 {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "bad_request",
			Status: http.StatusBadRequest,
			Err:    "no clients found",
		}, "no clients found", zap.String("service", service))
		return
	}

	if err = sh.Client.AddHookClient(clients, service); err != nil {
		appErr.HandleError(c, sh.Logger, appErr.HttpError{
			Code:   "conflict",
			Status: http.StatusConflict,
			Err:    err.Error(),
		}, "client already exists", zap.Error(err), zap.String("service", service))
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
// @Failure      400 {object} appErr.HttpError "Invalid service parameters"
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
// @Failure      400 {object} appErr.HttpError "Invalid service parameters"
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
			appErr.HandleError(c, sh.Logger, appErr.HttpError{
				Code:   "not_found",
				Status: http.StatusNotFound,
				Err:    err.Error(),
			}, "client not found", zap.Error(err), zap.String("service", service), zap.String("ip", t))
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

func buildDeltaBundle(ctx context.Context, patch *usecase.Patch, patchPath, tarGzPath string) error {

	buf := new(bytes.Buffer)
	err := utils.EncodeJson(buf, patch)
	if err != nil {
		return fmt.Errorf("%s: %w", appErr.ErrEncodeData.Error(), err)
	}

	if err := utils.SaveToFileWithLock(ctx, buf, patchPath); err != nil {
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
	if err := utils.SaveToFileWithLock(ctx, buf, dataPath); err != nil {
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

func getDataJson(dataPath string) (*usecase.Data, error) {
	byteOldData, err := os.ReadFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "failed to read data.json", err)
	}

	var oldData usecase.Data
	if err := json.Unmarshal(byteOldData, &oldData); err != nil {
		return nil, fmt.Errorf("%s: %w", "failed to unmarshal data to map", err)
	}

	return &oldData, nil
}
