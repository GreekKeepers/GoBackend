package requests

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

type Login struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type CreateReferalLink struct {
	Name string `json:"name"`
}

type RegisterUser struct {
	Login       string  `json:"login"`
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	ReferalLink *string `json:"referal_link"`
}

type WSrequest struct {
	Method string          `json:"method"`
	Id     uint            `json:"id"`
	Data   json.RawMessage `json:"data"`
}

type Bet struct {
	Amount   decimal.Decimal `json:"amount"`
	NumGames uint64          `json:"num_games"`
	UUID     string          `json:"-"`
	Data     string          `json:"data"`
	GameID   uint            `json:"game_id"`
	UserID   uint            `json:"-"`
	CoinID   uint            `json:"coin_id"`
	StopLoss decimal.Decimal `json:"stop_loss"`
	StopWin  decimal.Decimal `json:"stop_win"`
}

type ContinueGame struct {
	UUID   string `json:"-"`
	Data   string `json:"data"`
	GameID uint   `json:"game_id"`
	UserID uint   `json:"-"`
	CoinID uint   `json:"coin_id"`
}
type GetState struct {
	GameID uint `json:"game_id"`
	CoinID uint `json:"coin_id"`
}
