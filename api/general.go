package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) GetLeaderBoard(context *gin.Context) {
	leaderboardType := context.Param("type")
	timeBoundaries := context.Param("timeBoundaries")

	if leaderboardType == "" {
		slog.Error("No leaderboard present")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "No leaderboard present"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	if timeBoundaries == "" {
		slog.Error("No time boundaries present")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "No time boundaries present"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	if leaderboardType == "volume" {
		leaderboard, err := c.Db.FetchLeaderboardVolume(timeBoundaries)
		if err != nil {
			slog.Error("Getting leadeboard by volume", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting leaderboard"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
		response, _ := json.Marshal(leaderboard)
		context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})

	} else if leaderboardType == "profit" {
		leaderboard, err := c.Db.FetchLeaderboardProfit(timeBoundaries)
		if err != nil {
			slog.Error("Getting leadeboard by profit", "err", err)
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Error getting leaderboard"})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
		response, _ := json.Marshal(leaderboard)
		context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
	} else {
		slog.Error("Uknown leaderboard type")
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: "Uknown leaderboard type"})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}
	return
}

func GeneralEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.GET("/general/leaderboard/:type/:timeBoundaries", sCtrl.GetLeaderBoard)
}
