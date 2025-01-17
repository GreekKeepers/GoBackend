package responses

import (
	"time"

	"github.com/shopspring/decimal"
)

type Status string

const (
	Ok  = "OK"
	Err = "ERR"
)

type JsonResponse[T any] struct {
	Status Status `json:"status"`
	Data   T      `json:"data"`
}

type ErrorMessage struct {
	Message string `json:"message"`
}

// OK responses

type Ping struct {
	Pong string `json:"pong"`
}

type Credentials struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint64 `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type User struct {
	ID               uint      `json:"id"`
	RegistrationTime time.Time `json:"registration_time"`
	Username         string    `json:"username"`
	UserLevel        int64     `json:"user_level"`
}

type WSresponse struct {
	Id   uint        `json:"id"`
	Data interface{} `json:"data"`
}

type Bet struct {
	ID           uint            `json:"id"`
	Timestamp    time.Time       `json:"timestamp"`
	Amount       decimal.Decimal `json:"amount"`
	Profit       decimal.Decimal `json:"profit"`
	NumGames     int             `json:"num_games"`
	Outcomes     string          `json:"outcomes"`
	Profits      string          `json:"profits"`
	BetInfo      string          `json:"bet_info"`
	State        string          `json:"state"`
	UUID         string          `json:"uuid"`
	GameID       uint            `json:"game_id"`
	UserID       uint            `json:"user_id"`
	Username     string          `json:"username"`
	CoinID       uint            `json:"coin_id"`
	UserSeedID   uint            `json:"user_seed_id"`
	ServerSeedID uint            `json:"server_seed_id"`
}

type Leaderboard struct {
	UserId   uint            `gorm:"user_id" json:"user_id"`
	Total    decimal.Decimal `gorm:"total" json:"total"`
	Username string          `gorm:"username" json:"username"`
}
