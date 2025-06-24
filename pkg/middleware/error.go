package middleware

import (
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"

	"github.com/gin-gonic/gin"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			if httpErr, ok := err.Err.(*appErr.HttpError); ok {
				c.AbortWithStatusJSON(httpErr.Status, appErr.HttpError{
					Code:   httpErr.Code,
					Err:    httpErr.Err,
					Status: httpErr.Status,
				})
				return
			}
		}
	}
}
