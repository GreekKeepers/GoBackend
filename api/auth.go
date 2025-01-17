package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"greekkeepers.io/backend/auth"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) Login(context *gin.Context) {
	var submittedCredentials requests.Login

	if err := context.BindJSON(&submittedCredentials); err != nil {
		slog.Error("Parsing login data error", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	hashedPassword := auth.HashPassword(submittedCredentials.Password, c.Env.PasswordSalt)

	var existingUser db.User
	result := c.Db.Where("login = $1 AND password = $2", submittedCredentials.Login, hashedPassword).First(&existingUser)
	if result.Error != nil {
		slog.Error("No user found", "err", result.Error)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Wrong login or password"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	credentials, err := auth.CreateCredentials(strconv.FormatInt(int64(existingUser.ID), 10), "local", c.Env.RefreshTokenValidity, c.Env.RefreshTokenValidity, []byte(c.Env.PasswordSalt))
	if err != nil {
		slog.Error("Error issuing tokens", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error issuing tokens"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	refreshToken := &db.RefreshToken{
		Token:  credentials.RefreshToken,
		UserID: existingUser.ID,
	}
	if err := c.Db.Create(refreshToken).Error; err != nil {
		slog.Error("Error adding refresh token to db", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error creating refresh token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	token, err := json.Marshal(credentials)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: token})

}

func (c *SharedController) Refresh(context *gin.Context) {
	tokenString := context.Param("token")

	claims, err := auth.VerifyToken(tokenString, []byte(c.Env.PasswordSalt))
	if err != nil {
		slog.Error("Error verifying token", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Could not verify token"})
		context.IndentedJSON(http.StatusUnauthorized,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	aud, err := claims.GetAudience()
	if err != nil {
		slog.Error("Unable to get audience claims", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting audience claims"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	uuid, err := claims.GetSubject()
	if err != nil {
		slog.Error("Unable to get subject", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting audience claims"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if aud[0] != "refresh" {
		slog.Error("Not a refresh token")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Malformed token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	user_id, err := strconv.ParseUint(uuid, 10, 32)
	if err != nil {
		slog.Error("Unable to convert user id", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "bad user id"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	err = c.Db.Where("token = $1 AND user_id = $2", tokenString, user_id).Delete(&db.RefreshToken{}).Error
	if err != nil {
		slog.Error("Could not revoke token", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error revoking token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	credentials, err := auth.CreateCredentials(uuid, "local", c.Env.RefreshTokenValidity, c.Env.RefreshTokenValidity, []byte(c.Env.PasswordSalt))
	if err != nil {
		slog.Error("Error issuing tokens", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error issuing tokens"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	refreshToken := db.RefreshToken{
		Token:  credentials.RefreshToken,
		UserID: uint(user_id),
	}
	if err := c.Db.Create(refreshToken).Error; err != nil {
		slog.Error("Error adding refresh token to db", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error creating refresh token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	token, err := json.Marshal(credentials)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: token})
}

func (c *SharedController) Logout(context *gin.Context) {
	tokenString := context.Param("token")

	claims, err := auth.VerifyToken(tokenString, []byte(c.Env.PasswordSalt))
	if err != nil {
		slog.Error("Error verifying token", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Could not verify token"})
		context.IndentedJSON(http.StatusUnauthorized,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	aud, err := claims.GetAudience()
	if err != nil {
		slog.Error("Unable to get audience claims", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting audience claims"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	uuid, err := claims.GetSubject()
	if err != nil {
		slog.Error("Unable to get subject", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting audience claims"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if aud[0] != "refresh" {
		slog.Error("Not a refresh token")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Malformed token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	user_id, err := strconv.ParseUint(uuid, 10, 32)
	if err != nil {
		slog.Error("Unable to convert user id", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "bad user id"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	err = c.Db.Where("token = $1 AND user_id = $2", tokenString, user_id).Delete(&db.RefreshToken{}).Error
	if err != nil {
		slog.Error("Could not revoke token", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error revoking token"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	ok, err := json.Marshal("Token has been revoken")
	if err != nil {
		slog.Error("Serialization error", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: ok})
}

func (c *SharedController) Register(context *gin.Context) {
	var submittedCredentials requests.RegisterUser
	if err := context.BindJSON(&submittedCredentials); err != nil {
		slog.Error("Login Error", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if submittedCredentials.Username == "" {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "username is required"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	pattern := `^[a-zA-Z0-9_]+$`
	re := regexp.MustCompile(pattern)
	if !re.MatchString(submittedCredentials.Username) {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "bad username format"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if submittedCredentials.Login == "" {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "login is required"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	if !re.MatchString(submittedCredentials.Login) {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "bad login format"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if submittedCredentials.Password == "" {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "password is required"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	if len(submittedCredentials.Password) < 6 {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "password is too short"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	var existingUser db.User
	if err := c.Db.Where("login = ?", submittedCredentials.Login).First(&existingUser).Error; err == nil {
		slog.Error("User already exists", "Username", submittedCredentials.Username)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "User already exists"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	} else if err != gorm.ErrRecordNotFound {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Registration error"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	hashedPassword := auth.HashPassword(submittedCredentials.Password, c.Env.PasswordSalt)

	user := &db.User{
		Login:     submittedCredentials.Login,
		Username:  submittedCredentials.Username,
		Password:  hashedPassword,
		Provider:  "local",
		UserLevel: 0,
	}

	err := c.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("login = ?", submittedCredentials.Login).First(&existingUser).Error; err == nil {
			slog.Error("User already exists", "Username", submittedCredentials.Username)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "User already exists"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		} else if err != gorm.ErrRecordNotFound {
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Registration error"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}

		if err := tx.Create(user).Error; err != nil {
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Registration error"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}

		amount := db.Amount{
			UserID: user.ID,
			CoinID: 2,
			Amount: decimal.Zero,
		}
		if err := tx.Create(&amount).Error; err != nil {
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Amount creation error"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}

		amount = db.Amount{
			UserID: user.ID,
			CoinID: 1,
			Amount: decimal.New(1000, 0),
		}
		if err := tx.Create(&amount).Error; err != nil {
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Amount creation error"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}

		if submittedCredentials.ReferalLink != nil && *submittedCredentials.ReferalLink != "" {
			referalLink := db.ReferalLink{}
			if err := tx.Where("link_name = ?", *submittedCredentials.ReferalLink).First(&referalLink).Error; err != nil {
				slog.Error("unknown referal", "ref", *submittedCredentials.ReferalLink, "err", err)
				return nil
			}
			referal := db.Referal{
				ReferTo:   referalLink.ReferTo,
				ReferName: referalLink.ID,
				Referal:   user.ID,
			}
			if err := tx.Create(&referal).Error; err != nil {
				slog.Error("Referal creating", "err", err)
				var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Registration error"})
				context.IndentedJSON(http.StatusInternalServerError,
					responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
				return err
			}
		}
		return nil
	},
	)
	if err != nil {
		return
	}

	context.IndentedJSON(http.StatusOK, responses.JsonResponse[string]{Status: responses.Ok, Data: "User was created"})
}

func AuthEndpoints(sCtrl *SharedController, router *gin.Engine) {
	// auth endpoints
	router.POST("/login", sCtrl.Login)
	router.POST("/register", sCtrl.Register)
	router.GET("/refresh/:token", sCtrl.Refresh)
	router.DELETE("/logout/:token", AuthMiddleware(), sCtrl.Logout)
}
