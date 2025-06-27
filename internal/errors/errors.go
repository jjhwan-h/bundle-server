package errors

import (
	"errors"

	"github.com/gin-gonic/gin"
	contextkey "github.com/jjhwan-h/bundle-server/api/context"
	"go.uber.org/zap"
)

var (
	ErrEmptyEnvVar           = errors.New("environment variable is empty")
	ErrBuildData             = errors.New("failed to build data.json")
	ErrEncodeData            = errors.New("failed to encoding data")
	ErrSaveData              = errors.New("failed to save data")
	ErrBuildBundle           = errors.New("failed to build bundle")
	ErrNoChanges             = errors.New("no changes detected")
	ErrSendEventNotification = errors.New("send event notification failed")
	ErrAlreadyRegistered     = errors.New("client already registered")
)

func HandleError(c *gin.Context, logger *zap.Logger, httpErr HttpError, msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
	c.Set(contextkey.LogLevel, zap.ErrorLevel)
	c.Error(NewHttpError(
		httpErr.Code,
		httpErr.Status,
		httpErr.Err,
	))
}
