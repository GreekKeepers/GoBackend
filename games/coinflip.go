package games

import (
	"encoding/json"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type CoinFlipData struct {
	IsHeads bool `json:"is_heads"`
}

type CoinFlip struct {
	ProfitCoef decimal.Decimal `json:"profit_coef"`
}

func (g *CoinFlip) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := CoinFlipData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	profit := bet.Amount.Mul(g.ProfitCoef)

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	for game, number := range randomNumbers {
		side := number % 2
		outcomes[game] = side

		if (data.IsHeads && side == 1) || (!data.IsHeads && side == 0) {
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

func (*CoinFlip) NumbersPerBet() uint64 {
	return 1
}
