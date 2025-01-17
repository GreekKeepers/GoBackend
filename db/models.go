package db

import (
	"time"

	"github.com/shopspring/decimal"
)

type OAuthProvider string

const (
	Local    OAuthProvider = "local"
	Google   OAuthProvider = "google"
	Facebook OAuthProvider = "facebook"
	Twitter  OAuthProvider = "twitter"
)

type User struct {
	ID               uint          `gorm:"primaryKey"`
	RegistrationTime time.Time     `gorm:"autoCreateTime"`
	Login            string        `gorm:"unique;not null"`
	Username         string        `gorm:"not null"`
	Password         string        `gorm:"size:128;not null"`
	Provider         OAuthProvider `gorm:"type:oauth_provider;default:'local'"`
	UserLevel        int64         `gorm:"default:1"`
}

type RefreshToken struct {
	Token        string    `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null;"`
	User         User      `gorm:"not null;constraint:OnDelete:CASCADE"`
	CreationDate time.Time `gorm:"autoCreateTime"`
}

type Coin struct {
	ID    uint            `gorm:"primaryKey"`
	Name  string          `gorm:"unique;not null"`
	Price decimal.Decimal `gorm:"type:numeric(1000,4);not null"`
}

type Amount struct {
	UserID uint            `gorm:"not null;" json:"user_id"`
	User   User            `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	CoinID uint            `gorm:"not null;" json:"coin_id"`
	Coin   Coin            `gorm:"not null;constraint:OnDelete:CASCADE" json:"coin"`
	Amount decimal.Decimal `gorm:"type:numeric(1000,4);default:0" json:"amount"`
}
type Game struct {
	ID         uint   `gorm:"primaryKey"`
	Name       string `gorm:"unique;not null"`
	Parameters string `gorm:"not null"`
}

type UserSeed struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;" json:"user_id"`
	User      User      `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	UserSeed  string    `gorm:"size:64;not null" json:"seed"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type ServerSeed struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"not null;" json:"user_id"`
	User       User      `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	ServerSeed string    `gorm:"size:64;not null" json:"seed"`
	Revealed   bool      `gorm:"not null" json:"revealed"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Bet struct {
	ID        uint            `gorm:"primaryKey"`
	Timestamp time.Time       `gorm:"autoCreateTime"`
	Amount    decimal.Decimal `gorm:"type:numeric(1000,4)"`
	Profit    decimal.Decimal `gorm:"type:numeric(1000,4)"`
	NumGames  int             `gorm:"not null"`
	Outcomes  string          `gorm:"not null"`
	Profits   string          `gorm:"not null"`
	BetInfo   string          `gorm:"not null"`
	State     string
	UUID      string `gorm:"not null"`

	GameID       uint `gorm:"not null"`
	Game         Game `gorm:"not null;constraint:OnDelete:CASCADE"`
	UserID       uint `gorm:"not null;"`
	User         User `gorm:"not null;constraint:OnDelete:CASCADE"`
	CoinID       uint `gorm:"not null;"`
	Coin         Coin `gorm:"not null;constraint:OnDelete:CASCADE"`
	UserSeedID   uint `gorm:"not null;"`
	UserSeed     Coin `gorm:"not null;constraint:OnDelete:CASCADE"`
	ServerSeedID uint `gorm:"not null;"`
	ServerSeed   Coin `gorm:"not null;constraint:OnDelete:CASCADE"`
}

type Payout struct {
	ID             uint            `gorm:"primaryKey"`
	Timestamp      time.Time       `gorm:"autoCreateTime"`
	Amount         decimal.Decimal `gorm:"type:numeric(1000,4)"`
	Status         int             `gorm:"default:0"`
	AdditionalData string          `gorm:"not null"`
	UserID         uint            `gorm:"not null"`
	User           User            `gorm:"not null;constraint:OnDelete:CASCADE"`
}

type GameResult struct {
	TotalProfit decimal.Decimal
	Outcomes    []uint64
	Profits     []decimal.Decimal
	NumGames    uint32
	Data        string
	Finished    bool
}

type GameState struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Timestamp time.Time       `gorm:"autoCreateTime" json:"timestamp"`
	Amount    decimal.Decimal `gorm:"type:numeric(1000,4)" json:"amount"`
	BetInfo   string          `gorm:"not null" json:"bet_info"`
	State     string          `gorm:"not null" json:"state"`
	UUID      string          `gorm:"not null" json:"uuid"`

	GameID       uint `gorm:"not null" json:"game_id"`
	Game         Game `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	UserID       uint `gorm:"not null;" json:"user_id"`
	User         User `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	CoinID       uint `gorm:"not null;" json:"coin_id"`
	Coin         Coin `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	UserSeedID   uint `gorm:"not null;" json:"user_seed_id"`
	UserSeed     Coin `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
	ServerSeedID uint `gorm:"not null;" json:"server_seed_id"`
	ServerSeed   Coin `gorm:"not null;constraint:OnDelete:CASCADE" json:"-"`
}

type ReferalLink struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	ReferTo  uint   `gorm:"not null;unique;constraint:OnDelete:CASCADE;references:User(ID)"`
	LinkName string `gorm:"size:8;not null;unique"`
}

type Referal struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	ReferTo    uint      `gorm:"not null;constraint:OnDelete:CASCADE;references:User(ID)"`
	ReferName  uint      `gorm:"not null;constraint:OnDelete:CASCADE;references:ReferalLink(ID)"`
	Referal    uint      `gorm:"not null;constraint:OnDelete:CASCADE;references:User(ID)"`
	CreateDate time.Time `gorm:"autoCreateTime"`
}
