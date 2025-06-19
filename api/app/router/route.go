package router

import (
	"bundle-server/pkg/middleware"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	contextTime = 3
)

func New(logger *zap.Logger) (*gin.Engine, error) {
	r := gin.New()

	timeout := time.Duration(contextTime) * time.Second

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.Security())
	}
	r.Use(middleware.ErrorMiddleware())

	r.Group("/data")
	{
		err := NewDataRouter(r, logger, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize data router: %w", err)
		}
	}

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	return r, nil
}
