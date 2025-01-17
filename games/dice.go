package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

var (
	DICE_LOWER_BOUNDARY decimal.Decimal
	DICE_UPPER_BOUNDARY decimal.Decimal
	DICE_MULT           decimal.Decimal
	U64_UPPER_BOUNDARY  decimal.Decimal
	HUNDRED             decimal.Decimal
	NINTYNINE           decimal.Decimal
)

func init() {
	DICE_LOWER_BOUNDARY, _ = decimal.NewFromString("1.0421")
	DICE_UPPER_BOUNDARY, _ = decimal.NewFromString("99.9999")
	DICE_MULT, _ = decimal.NewFromString("10000")
	U64_UPPER_BOUNDARY, _ = decimal.NewFromString("18446744073709551615")
	HUNDRED, _ = decimal.NewFromString("100")
	NINTYNINE, _ = decimal.NewFromString("99")
}

type DiceData struct {
	RollOver   bool            `json:"roll_over"`
	Multiplier decimal.Decimal `json:"multiplier"`
}

// Dice contains constants for dice boundaries.
type Dice struct {
	LowerBoundary decimal.Decimal `json:"lower_boundary"`
	UpperBoundary decimal.Decimal `json:"upper_boundary"`
}

func remap(
	number decimal.Decimal,
	from decimal.Decimal,
	to decimal.Decimal,
	map_from decimal.Decimal,
	map_to decimal.Decimal,
) decimal.Decimal {
	a := number.Sub(from)
	b := to.Sub(from)
	c := a.Div(b)

	d := map_to.Sub(map_from)

	c = c.Mul(d)

	return c.Add(map_from)
}

func (g *Dice) Play(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := DiceData{}
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

		if data.RollOver && number.GreaterThanOrEqual(numberToRoll) || !data.RollOver && numberToRoll.GreaterThanOrEqual(number) {
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

func (*Dice) NumbersPerBet() uint64 {
	return 1
}
