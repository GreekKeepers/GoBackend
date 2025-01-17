package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) ListCoins(context *gin.Context) {

	var coins []db.Coin

	c.Db.Find(&coins)

	response, _ := json.Marshal(coins)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func CoinEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.GET("/coin/list", sCtrl.ListCoins)
}
