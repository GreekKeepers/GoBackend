package games

import (
	"encoding/json"
	"errors"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type ApplesData struct {
	Difficulty uint8 `json:"difficulty"`
}

type ApplesDifficulty struct {
	Mines       uint8 `json:"mines"`
	TotalSpaces uint8 `json:"total_spaces"`
}

type ApplesState struct {
	State             [][]bool        `json:"state"`
	PickedTiles       []uint8         `json:"picked_tiles"`
	CurrentMultiplier decimal.Decimal `json:"current_multiplier"`
}

type ApplesContinueData struct {
	Tile    uint8 `json:"tile"`
	Cashout bool  `json:"cashout"`
}

type Apples struct {
	Difficulties []ApplesDifficulty  `json:"difficulties"`
	Multipliers  [][]decimal.Decimal `json:"multipliers"`
}

func (g *Apples) StartPlaying(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	data := ApplesData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}

	if int(data.Difficulty) >= len(g.Difficulties) {
		return db.GameResult{}, errors.New("Bad difficulty")
	}

	returnData, _ := json.Marshal(ApplesState{
		State:             [][]bool{},
		PickedTiles:       []uint8{},
		CurrentMultiplier: decimal.Zero,
	})

	return db.GameResult{
		TotalProfit: decimal.Zero,
		Outcomes:    []uint64{},
		Profits:     []decimal.Decimal{},
		NumGames:    uint32(1),
		Data:        string(returnData),
		Finished:    false,
	}, nil
}

func (g *Apples) ContinuePlaying(state db.GameState, bet requests.ContinueGame, randomNumbers []uint64) (db.GameResult, error) {
	data := ApplesContinueData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}
	parsedState := ApplesState{}
	err = json.Unmarshal([]byte(state.State), &parsedState)
	if err != nil {
		return db.GameResult{}, err
	}
	initialData := ApplesData{}
	err = json.Unmarshal([]byte(state.BetInfo), &parsedState)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.Cashout && parsedState.CurrentMultiplier.IsZero() {
		profit := state.Amount.Mul(parsedState.CurrentMultiplier)

		return db.GameResult{
			TotalProfit: profit,
			Outcomes:    make([]uint64, len(parsedState.State)),
			Profits:     []decimal.Decimal{profit},
			NumGames:    1,
			Data:        state.State,
			Finished:    true,
		}, nil
	}

	difficulty := g.Difficulties[initialData.Difficulty]

	pickedTile := data.Tile
	if pickedTile >= difficulty.TotalSpaces {
		return db.GameResult{}, errors.New("Picked tile is larger than allowed amount")
	}

	row := make([]bool, difficulty.TotalSpaces)

	rng := randomNumbers[0]

	if difficulty.Mines == 1 {
		mineIndex := rng % uint64(difficulty.TotalSpaces)
		row[mineIndex] = true
	} else {
		for i, _ := range row {
			row[i] = true
		}

		emptyIndex := rng % uint64(difficulty.TotalSpaces)
		row[emptyIndex] = false
	}

	won := !row[pickedTile]

	newState := append(parsedState.State, row)
	parsedState.State = newState

	newPickedTiles := append(parsedState.PickedTiles, pickedTile)
	parsedState.PickedTiles = newPickedTiles

	if won {
		parsedState.CurrentMultiplier = g.Multipliers[initialData.Difficulty][len(parsedState.State)-1]
		profit := g.Multipliers[initialData.Difficulty][len(parsedState.State)-1].Mul(state.Amount)
		stringState, _ := json.Marshal(parsedState)
		return db.GameResult{
			TotalProfit: profit,
			Outcomes:    make([]uint64, len(parsedState.State)),
			Profits:     []decimal.Decimal{profit},
			NumGames:    1,
			Data:        string(stringState),
			Finished:    len(parsedState.State) == 9,
		}, nil
	} else {
		parsedState.CurrentMultiplier = decimal.Zero
		stringState, _ := json.Marshal(parsedState)
		return db.GameResult{
			TotalProfit: decimal.Zero,
			Outcomes:    make([]uint64, len(parsedState.State)),
			Profits:     []decimal.Decimal{decimal.Zero},
			NumGames:    1,
			Data:        string(stringState),
			Finished:    true,
		}, nil
	}
}

func (*Apples) NumbersPerBet() uint64 {
	return 1
}
