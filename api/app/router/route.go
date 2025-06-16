package router

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	contextTime = 3
)

func New() (*gin.Engine, error) {
	r := gin.Default()

	timeout := time.Duration(contextTime) * time.Second

	r.Use(ErrorMiddleware())

	r.Group("/data")
	{
		err := NewDataRouter(r, timeout)
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
