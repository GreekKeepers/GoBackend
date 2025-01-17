package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type WheelData struct {
	Risk       uint32 `json:"risk"`
	NumSectors uint32 `json:"num_sectors"`
}

type Wheel struct {
	Multipliers   [][][]decimal.Decimal `json:"multipliers"`
	MaxRisk       uint32                `json:"max_risk"`
	MaxNumSectors uint32                `json:"max_num_sectors"`
}

func (g *Wheel) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := WheelData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}
	if data.Risk > g.MaxRisk || data.NumSectors > g.MaxNumSectors {
		return db.GameResult{}, errors.New("Bad input")
	}

	multipliers := g.Multipliers[data.Risk][data.NumSectors]

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	numSectors := uint64((data.NumSectors + 1) * 10)

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	for game, number := range randomNumbers {
		sector := number % numSectors
		outcomes[game] = sector

		multiplier := multipliers[sector]

		if multiplier.IsZero() {
			totalValue = totalValue.Sub(bet.Amount)
			profits[game] = decimal.Zero

		} else {
			profit := bet.Amount.Mul(multiplier)
			totalProfit = totalProfit.Add(profit)
			totalValue = totalProfit.Add(profit)
			profits[game] = profit
		}

		games += 1

		if (!bet.StopWin.IsZero() && totalValue.GreaterThanOrEqual(bet.StopWin)) || (!bet.StopLoss.IsZero() && totalValue.LessThanOrEqual(bet.StopLoss)) {
			break
		}
	}
	if games != bet.NumGames {
		totalProfit = totalProfit.Add(decimal.NewFromUint64(bet.NumGames - games))
	}

	return db.GameResult{
		TotalProfit: totalProfit,
		Outcomes:    outcomes[0:games],
		Profits:     profits[0:games],
		NumGames:    uint32(games),
		Data:        bet.Data,
		Finished:    true,
	}, nil
}

func (*Wheel) NumbersPerBet() uint64 {
	return 1
}
