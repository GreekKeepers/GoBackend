package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type RPSData struct {
	Action uint64 `json:"action"` // 0 - rock, 1 - paper, 2 - scissors
}

type RPS struct {
	ProfitCoef decimal.Decimal `json:"profit_coef"`
	DrawCoef   decimal.Decimal `json:"draw_coef"`
}

func rpsOutcome(player uint64, rng uint64) uint32 {
	if player == rng {
		return 2
	}
	if player == 0 {
		if rng == 1 {
			return 0
		} else {
			return 1
		}
	}

	if player == 1 {
		if rng == 2 {
			return 0
		} else {
			return 1
		}
	}

	if player == 2 {
		if rng == 0 {
			return 0
		} else {
			return 1
		}
	}

	panic("Bad input")
}

func (g *RPS) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := RPSData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.Action > 2 {
		return db.GameResult{}, errors.New("Bad action")
	}

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	profit := bet.Amount.Mul(g.ProfitCoef)
	draw := bet.Amount.Mul(g.DrawCoef)

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	for game, number := range randomNumbers {
		action := number % 3
		outcomes[game] = action

		rpsResult := rpsOutcome(data.Action, action)

		if rpsResult == 2 {
			totalProfit = totalProfit.Add(draw)
			totalValue = totalProfit.Add(draw)
			profits[game] = draw
		} else if rpsResult == 1 {
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

func (*RPS) NumbersPerBet() uint64 {
	return 1
}
