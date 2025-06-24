package router

import (
	"fmt"
	"time"

	"github.com/jjhwan-h/bundle-server/config"
	"github.com/jjhwan-h/bundle-server/pkg/middleware"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New(logger *zap.Logger) (*gin.Engine, error) {
	r := gin.New()

	timeout := time.Duration(config.Cfg.HTTP.ContextTime) * time.Second

	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))
	if gin.Mode() == gin.ReleaseMode {
		r.Use(middleware.Security())
	}
	r.Use(middleware.ErrorMiddleware())

	r.Group("")
	{
		err := NewServiceRouter(r, logger, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize data router: %w", err)
		}
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	return r, nil
}
