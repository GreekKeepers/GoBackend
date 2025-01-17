package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"math/rand/v2"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/blake2b"
	"gorm.io/gorm"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) GetUser(context *gin.Context) {
	userId := context.Param("userID")

	var userFull db.User
	if err := c.Db.Where("id = ?", userId).First(&userFull).Error; err != nil {
		slog.Error("User not found", "userId", userId)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "User not found"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	user := responses.User{
		ID:               userFull.ID,
		RegistrationTime: userFull.RegistrationTime,
		Username:         userFull.Username,
		UserLevel:        userFull.UserLevel,
	}

	response, _ := json.Marshal(user)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) GetUserAmounts(context *gin.Context) {
	userId := context.Param("userID")

	var amounts []db.Amount
	if err := c.Db.Preload("Coin").Where("user_id = ?", userId).Find(&amounts).Error; err != nil {
		slog.Error("User not found", "userId", userId)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "User not found"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	response, _ := json.Marshal(amounts)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) GetLatestGames(context *gin.Context) {
	userID, err := strconv.Atoi(context.Param("userID"))
	if err != nil {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	var games []string
	if err := c.Db.Raw(`
		SELECT games.name FROM games RIGHT JOIN 
                (SELECT * from bets where bets.user_id=$1 ORDER BY timestamp DESC LIMIT 2) as bets ON bets.game_id = games.id
		`, userID,
	).Scan(&games).Error; err != nil {
		slog.Error("Games not found", "userId", userID)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "User not found"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	response, _ := json.Marshal(games)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) SetUserSeed(context *gin.Context) {
	userSeed := context.Param("newSeed")

	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	hashed := blake2b.Sum256([]byte(userSeed))
	hashedSeed := hex.EncodeToString(hashed[:])
	seed := &db.UserSeed{
		UserID:   uint(userId),
		UserSeed: hashedSeed,
	}

	if err := c.Db.Create(seed).Error; err != nil {
		slog.Error("Failed adding a seed", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Adding a seed failed"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	response, _ := json.Marshal("Seed was added")
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) GetUserSeed(context *gin.Context) {
	seedId := context.Param("seedId")

	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	seed := &db.UserSeed{}
	if seedId == "" || seedId == "0" {
		err := c.Db.Where("user_id = ?", userId).Order("created_at DESC").First(seed).Error
		if err != nil {
			slog.Error("Error getting user seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting user seed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
	} else {
		seedIdNum, err := strconv.ParseUint(seedId, 10, 32)
		if err != nil {
			fmt.Println(err)
		}

		err = c.Db.Where("user_id = ? AND id = ?", userId, seedIdNum).Order("created_at DESC").First(seed).Error
		if err != nil {
			slog.Error("Error getting user seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting user seed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
	}
	response, _ := json.Marshal(seed)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) GetServerSeed(context *gin.Context) {
	seedId := context.Param("seedId")

	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	seed := &db.ServerSeed{}
	if seedId == "" || seedId == "0" {
		err := c.Db.Where("user_id = ?", userId).Order("created_at DESC").First(seed).Error
		if err != nil {
			slog.Error("Error getting server seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting server seed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
	} else {
		seedIdNum, err := strconv.ParseUint(seedId, 10, 32)
		if err != nil {
			fmt.Println(err)
		}

		err = c.Db.Where("user_id = ? AND id = ?", userId, seedIdNum).Order("created_at DESC").First(seed).Error
		if err != nil {
			slog.Error("Error getting server seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting server seed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
	}
	response, _ := json.Marshal(seed)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) NewServerSeed(context *gin.Context) {
	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	randomNumber := strconv.FormatInt(rand.Int64N(1000000000000000000), 10) + c.Env.PasswordSalt + strconv.FormatInt(rand.Int64N(1000000000000000000), 10)

	hashed := blake2b.Sum256([]byte(randomNumber))
	hashedSeed := hex.EncodeToString(hashed[:])
	seed := &db.ServerSeed{
		UserID:     uint(userId),
		ServerSeed: hashedSeed,
		Revealed:   false,
	}

	err = c.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&db.ServerSeed{}).Where("user_id=? AND revealed=FALSE", userId).Update("revealed", true).Error; err != nil {
			slog.Error("Failed adding a seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Adding a seed failed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}
		if err := tx.Create(seed).Error; err != nil {
			slog.Error("Failed adding a seed", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Adding a seed failed"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return err
		}
		return nil
	})
	if err != nil {
		return
	}

	response, _ := json.Marshal(hashedSeed)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func UserEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.GET("/user/userseed/:seedId", AuthMiddleware(), sCtrl.GetUserSeed)
	router.GET("/user/userseed", AuthMiddleware(), sCtrl.GetUserSeed)
	router.GET("/user/serverseed/:seedId", AuthMiddleware(), sCtrl.GetServerSeed)
	router.GET("/user/serverseed", AuthMiddleware(), sCtrl.GetServerSeed)
	router.GET("/user/:userID", sCtrl.GetUser)
	router.POST("/user/userseed/:newSeed", AuthMiddleware(), sCtrl.SetUserSeed)
	router.POST("/user/serverseed", AuthMiddleware(), sCtrl.NewServerSeed)
	router.GET("/user/amounts/:userID", sCtrl.GetUserAmounts)
	router.GET("/user/latest/:userID", sCtrl.GetLatestGames)

}
