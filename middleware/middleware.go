package middleware

import (
	"github.com/gin-gonic/gin"
)

func SetMiddlewareJSON() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("Content-Type", "application/json")
		ctx.Next()
	}
}
