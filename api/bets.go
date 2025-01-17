package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"greekkeepers.io/backend/responses"
)

func (c *SharedController) GetBets(context *gin.Context) {
	gameName := context.Param("gameName")
	if gameName == "" {
		var bets []responses.Bet

		c.Db.Raw(`
		SELECT 
                Bets.id,
                Bets.timestamp,
                Bets.amount,
                Bets.profit,
                Bets.num_games,
                Bets.bet_info,
                Bets.state,
                Bets.uuid,
                Bets.game_id,
                Bets.user_id,
                Users.username,
                Bets.coin_id,
                Bets.user_seed_id,
                Bets.server_seed_id,
                Bets.outcomes,
                Bets.profits
            FROM Bets
            INNER JOIN Users ON Bets.user_id=Users.id
            ORDER BY Bets.timestamp DESC
            LIMIT $2
	`, 10).Scan(&bets)

		response, _ := json.Marshal(bets)
		context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})

	}

	var bets []responses.Bet

	c.Db.Raw(`
		SELECT 
                Bets.id,
                Bets.timestamp,
                Bets.amount,
                Bets.profit,
                Bets.num_games,
                Bets.bet_info,
                Bets.state,
                Bets.uuid,
                Bets.game_id,
                Bets.user_id,
                Users.username,
                Bets.coin_id,
                Bets.user_seed_id,
                Bets.server_seed_id,
                Bets.outcomes,
                Bets.profits
            FROM Bets
            INNER JOIN Games ON Bets.game_id=Games.id
            INNER JOIN Users ON Bets.user_id=Users.id
            WHERE Games.name=$1
            ORDER BY Bets.timestamp DESC
            LIMIT $2
	`, gameName, 10).Scan(&bets)

	response, _ := json.Marshal(bets)
	context.IndentedJSON(http.StatusOK, responses.JsonResponse[json.RawMessage]{Status: responses.Ok, Data: response})
}

func (c *SharedController) GetUserBets(context *gin.Context) {
	strOffset := context.Query("offset")
	offset := 0
	if strOffset != "" {
		offset, err := strconv.Atoi(strOffset)
		if err != nil || offset < -1 {
			var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
			context.IndentedJSON(http.StatusInternalServerError,
				responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
			return
		}
	}

	userID, err := strconv.Atoi(context.Param("userID"))
	if err != nil {
		var err_msg, _ = json.Marshal(responses.ErrorMessage{Message: err.Error()})
		context.IndentedJSON(http.StatusInternalServerError,
			responses.JsonResponse[json.RawMessage]{Status: responses.Err, Data: err_msg})
		return
	}

	var bets []responses.Bet
	c.Db.Raw(`
	    SELECT 
                Bets.id,
                Bets.timestamp,
                Bets.amount,
                Bets.profit,
                Bets.num_games,
                Bets.bet_info,
                Bets.state,
                Bets.uuid,
                Bets.game_id,
                Bets.user_id,
                Users.username,
                Bets.coin_id,
                Bets.user_seed_id,
                Bets.server_seed_id,
                Bets.outcomes,
                Bets.profits
            FROM Bets
            INNER JOIN Users ON bets.user_id = Users.id
            WHERE bets.user_id = $1 
            ORDER BY Bets.id DESC
            LIMIT $2 
	    OFFSET $3
	`, userID, 10, offset).Scan(&bets)

}

func BetsEndpoints(sCtrl *SharedController, router *gin.Engine) {
	router.GET("/bets/list/:gameName", sCtrl.GetBets)
	router.GET("/bets/list", sCtrl.GetBets)
	router.GET("/bets/user/:userID", sCtrl.GetUserBets)
}
