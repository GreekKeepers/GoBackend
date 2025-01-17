package games

import (
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type StatelessGameEngine interface {
	Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error)
	NumbersPerBet() uint64
}
type StatefulGameEngine interface {
	StartPlaying(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error)
	ContinuePlaying(state db.GameState, bet requests.ContinueGame, randomNumbers []uint64) (db.GameResult, error)
	NumbersPerBet() uint64
}
