package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) CreateReferalLink(context *gin.Context) {
	var submittedReferalLink requests.CreateReferalLink

	if err := context.BindJSON(&submittedReferalLink); err != nil {
		slog.Error("Parsing login data error", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if submittedReferalLink.Name == "" {
		slog.Error("Empty link name submitted")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Empty link name submitted"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		slog.Error("Converting user id error", "err", err)
	}

	referalLink := db.ReferalLink{
		ReferTo:  uint(userId),
		LinkName: submittedReferalLink.Name,
	}

	if err := c.Db.Create(&referalLink).Error; err != nil {
		slog.Error("Parsing login data error", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	token, err := json.Marshal("Link was created")
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: token})
}

func (c *SharedController) GetReferalLink(context *gin.Context) {
	sub := context.GetString("uuid")
	if sub == "" {
		return
	}

	userId, err := strconv.ParseUint(sub, 10, 32)
	if err != nil {
		slog.Error("Converting user id error", "err", err)
	}

	referalLink := []db.ReferalLink{}

	if err := c.Db.Where("refer_to=?", userId).Find(&referalLink).Error; err != nil {
		slog.Error("Couldn't find links", "err", err)
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Couldn't find links"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	token, err := json.Marshal(referalLink)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: token})

}

func ReferalEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.POST("/ref", AuthMiddleware(), sCtrl.CreateReferalLink)
	router.GET("/ref", AuthMiddleware(), sCtrl.GetReferalLink)
}
