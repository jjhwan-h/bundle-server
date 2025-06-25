package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	contextkey "github.com/jjhwan-h/bundle-server/api/context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		path := c.Request.URL.Path

		fileds := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.RequestURI),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		}

		if strings.HasPrefix(path, "/swagger/") {
			logger.Debug("Swagger",
				fileds...)
			return
		}

		level := zap.InfoLevel
		if l, ok := c.Get(contextkey.LogLevel); ok {
			if parsed, ok := l.(zapcore.Level); ok {
				level = parsed
			}
		}

		switch level {
		case zap.DebugLevel:
			logger.Debug("Request", fileds...)
		case zap.InfoLevel:
			logger.Info("Request", fileds...)
		case zap.ErrorLevel:
			logger.Error("Request", fileds...)
		case zap.WarnLevel:
			logger.Warn("Request", fileds...)
		default:
			logger.Info("Request", fileds...)
		}
	}
}
