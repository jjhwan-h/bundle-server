package handler

import (
	"bundle-server/domain/usecase"
	appErr "bundle-server/internal/errors"
	"bundle-server/internal/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type DataHandler struct {
	CasbUsecase usecase.CasbUsecase
}

func (dh *DataHandler) BuildDataJson(c *gin.Context) {
	data, err := dh.CasbUsecase.BuildDataJson(c)
	if err != nil {
		log.Printf("[ERROR] %s: %v\n", appErr.ErrBuildData.Error(), err)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			fmt.Errorf("failed to build data.json"),
		))
	}

	/*data.json 저장*/
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		log.Printf("[ERROR] %s: %v\n", appErr.ErrEncodeData.Error(), err)
		c.Error(appErr.NewHttpError(
			"internal_server_error",
			http.StatusInternalServerError,
			fmt.Errorf("failed to encoding data"),
		))
	}

	go func() {
		if err := utils.SaveToFile(buf, fmt.Sprintf("%s/data.json", viper.GetString("OPA_DATA_PATH"))); err != nil {
			log.Printf("[ERROR] %s: %v", appErr.ErrSaveData.Error(), err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "data.json is being saved in the background",
	})
}
