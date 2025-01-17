package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

func init() {
	DICE_LOWER_BOUNDARY, _ = decimal.NewFromString("1.0421")
	DICE_UPPER_BOUNDARY, _ = decimal.NewFromString("99.9999")
	DICE_MULT, _ = decimal.NewFromString("10000")
	U64_UPPER_BOUNDARY, _ = decimal.NewFromString("18446744073709551615")
	HUNDRED, _ = decimal.NewFromString("100")
	NINTYNINE, _ = decimal.NewFromString("99")
}

type RocketData struct {
	Multiplier decimal.Decimal `json:"multiplier"`
}

// Dice contains constants for dice boundaries.
type Rocket struct {
	LowerBoundary decimal.Decimal `json:"lower_boundary"`
	UpperBoundary decimal.Decimal `json:"upper_boundary"`
}

func (g *Rocket) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := RocketData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.Multiplier.LessThan(DICE_LOWER_BOUNDARY) || data.Multiplier.GreaterThan(DICE_UPPER_BOUNDARY) {
		return db.GameResult{}, errors.New("roll_under value out of bounds")
	}

	totalProfit := decimal.Zero
	totalValue := decimal.Zero
	games := uint64(0)

	profit := bet.Amount.Mul(data.Multiplier)
	numberToRoll := HUNDRED.Sub(NINTYNINE.Div(data.Multiplier))

	outcomes := make([]uint64, len(randomNumbers))
	profits := make([]decimal.Decimal, len(randomNumbers))
	for game, number := range randomNumbers {
		number := remap(
			decimal.NewFromUint64(number),
			decimal.Zero,
			U64_UPPER_BOUNDARY,
			DICE_LOWER_BOUNDARY,
			DICE_UPPER_BOUNDARY,
		)
		outcomes[game] = number.Mul(DICE_MULT).BigInt().Uint64()

		if number.GreaterThanOrEqual(numberToRoll) {
			totalProfit = totalProfit.Add(profit)
			totalValue = totalValue.Add(profit)
			profits[game] = profit
		} else {
			totalValue = totalValue.Sub(bet.Amount)
			profits[game] = decimal.Zero
		}

		games++
		if (!bet.StopWin.IsZero() && totalValue.GreaterThanOrEqual(bet.StopWin)) || (!bet.StopLoss.IsZero() && totalValue.LessThanOrEqual(bet.StopLoss)) {
			break
		}
	}

	if games != bet.NumGames {
		totalProfit = totalProfit.Add(decimal.NewFromUint64(bet.NumGames - games))
	}

	return db.GameResult{
		TotalProfit: totalProfit,
		Outcomes:    outcomes[:games],
		Profits:     profits[:games],
		NumGames:    uint32(games),
		Data:        bet.Data,
		Finished:    true,
	}, nil
}

func (*Rocket) NumbersPerBet() uint64 {
	return 1
}
