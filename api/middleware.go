package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"greekkeepers.io/backend/auth"
	"greekkeepers.io/backend/responses"
)

func SetMiddlewareJSON() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("Content-Type", "application/json")
		ctx.Next()
	}
}

func Auth(context *gin.Context, secretKey string) (string, error) {

	tokenString := context.GetHeader("Authorization")
	if tokenString == "" {
		slog.Error("No token supplied")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Token is not present"})
		context.IndentedJSON(http.StatusUnauthorized,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return "", errors.New("Unauthorized")
	}

	fmt.Println(tokenString)

	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	claims, err := auth.VerifyToken(tokenString, []byte(secretKey))
	if err != nil {
		slog.Error("Error verifying token", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusUnauthorized,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return "", errors.New("Unauthorized")
	}

	sub, _ := claims.GetSubject()
	return sub, nil
}

func AuthMiddleware() gin.HandlerFunc {
	authSecretKey := os.Getenv("PASSWORD_SALT")
	return func(c *gin.Context) {
		uuid, err := Auth(c, authSecretKey)
		if err != nil {
			c.AbortWithStatus(401)
			return
		}
		c.Set("uuid", uuid)
		c.Next()
	}
}
