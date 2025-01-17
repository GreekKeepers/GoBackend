package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type RaceData struct {
	Car uint64 `json:"car"`
}

type Race struct {
	ProfitCoef decimal.Decimal `json:"profit_coef"`
	CarsAmount uint64          `json:"cars_amount"`
}

func (g *Race) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := RaceData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.Car >= g.CarsAmount {
		return db.GameResult{}, errors.New("Bad car")
	}

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	profit := bet.Amount.Mul(g.ProfitCoef)

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	for game, number := range randomNumbers {
		winnerCar := number % g.CarsAmount
		outcomes[game] = winnerCar

		if data.Car == winnerCar {
			totalProfit = totalProfit.Add(profit)
			totalValue = totalProfit.Add(profit)
			profits[game] = profit
		} else {
			totalValue = totalValue.Sub(profit)
			profits[game] = decimal.Zero
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

func (*Race) NumbersPerBet() uint64 {
	return 1
}
