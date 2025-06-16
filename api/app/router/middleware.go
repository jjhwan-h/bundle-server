package router

import (
	"context"
	"time"

	appErr "bundle-server/internal/errors"

	"github.com/gin-gonic/gin"
)

func TimeOutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			if httpErr, ok := err.Err.(*appErr.HttpError); ok {
				c.AbortWithStatusJSON(httpErr.Status, gin.H{
					"error":   httpErr.Code,
					"message": httpErr.Err.Error(),
				})
				return
			}
		}
	}
}
