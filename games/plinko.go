package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type PlinkoData struct {
	NumRows uint64 `json:"num_rows"`
	Risk    uint64 `json:"risk"`
}

type PlinkoReturnData struct {
	NumRows uint64    `json:"num_rows"`
	Risk    uint64    `json:"risk"`
	Paths   [][]uint8 `json:"paths"`
}

type Plinko struct {
	Multipliers [][][]decimal.Decimal
}

func (g *Plinko) plinkoGame(rng uint64, numRows uint64, risk uint64) (decimal.Decimal, []uint8) {
	result := make([]uint8, numRows)

	mask := uint64(0x8000000000000000)
	ended := int8(0)

	for i := range numRows {
		res := uint8(0)
		if rng%mask > 0 {
			ended += 1
			res = 1
		} else {
			ended -= 1
			res = 0
		}
		mask >>= 1
		result[i] = res
	}

	slot := (ended + int8(numRows)) >> 1
	multiplier := g.Multipliers[risk][numRows-8][slot]

	return multiplier, result
}

func (g *Plinko) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := PlinkoData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.NumRows < 8 || data.NumRows > 16 {
		return db.GameResult{}, errors.New("bad rows number")
	}
	if data.Risk >= 3 {
		return db.GameResult{}, errors.New("bad risk")
	}

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	paths := make([][]uint8, len(randomNumbers))
	for game, number := range randomNumbers {
		multiplier, path := g.plinkoGame(number, data.NumRows, data.Risk)
		payout := bet.Amount.Mul(multiplier)

		paths[game] = path
		profits[game] = payout
		games += 1
		outcomes[game] = number

		totalProfit = totalProfit.Add(payout)
		totalValue = totalValue.Add(payout.Sub(bet.Amount))

		if (!bet.StopWin.IsZero() && totalValue.GreaterThanOrEqual(bet.StopWin)) || (!bet.StopLoss.IsZero() && totalValue.LessThanOrEqual(bet.StopLoss)) {
			break
		}

	}
	if games != bet.NumGames {
		totalProfit = totalProfit.Add(decimal.NewFromUint64(bet.NumGames - games))
	}

	returnData := PlinkoReturnData{
		NumRows: data.NumRows,
		Risk:    data.Risk,
		Paths:   paths,
	}

	retData, err := json.Marshal(returnData)

	return db.GameResult{
		TotalProfit: totalProfit,
		Outcomes:    outcomes[0:games],
		Profits:     profits[0:games],
		NumGames:    uint32(games),
		Data:        string(retData),
		Finished:    true,
	}, nil
}

func (*Plinko) NumbersPerBet() uint64 {
	return 1
}
