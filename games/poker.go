package games

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/shopspring/decimal"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/requests"
)

type PokerData struct{}

type Card struct {
	Number uint8 `json:"number"`
	Suit   uint8 `json:"suit"`
}

type PokerState struct {
	CardsInHand []Card `json:"cards_in_hand"`
}

type PokerContinueData struct {
	Replace   bool `json:"replace"`
	ToReplace []bool
}

type Poker struct {
	InitialDeck []Card            `json:"initial_deck"`
	Multipliers []decimal.Decimal `json:"multipliers"`
}

func pickCard(rng uint64, deck *[]Card) Card {
	position := rng % uint64(len(*deck))
	card := (*deck)[position]

	(*deck)[position] = (*deck)[len(*deck)-1]
	newDeck := make([]Card, len(*deck)-1)

	for i := range len(*deck) - 1 {
		newDeck[i] = (*deck)[i]
	}

	return card
}

func (g *Poker) StartPlaying(bet requests.Bet, randomNumbers []uint64) (db.GameResult, error) {
	deck := make([]Card, len(g.InitialDeck))
	copy(deck, g.InitialDeck)
	cardsInHand := make([]Card, 5)
	for i := range 5 {
		card := pickCard(randomNumbers[i], &deck)
		cardsInHand[i] = card

	}

	data, _ := json.Marshal(PokerState{
		CardsInHand: cardsInHand,
	})

	return db.GameResult{
		TotalProfit: decimal.Zero,
		Outcomes:    randomNumbers,
		Profits: []decimal.Decimal{
			decimal.Zero,
		},
		NumGames: uint32(1),
		Data:     string(data),
		Finished: false,
	}, nil
}
func (g *Poker) ContinuePlaying(state db.GameState, bet requests.ContinueGame, randomNumbers []uint64) (db.GameResult, error) {
	data := PokerContinueData{}
	err := json.Unmarshal([]byte(bet.Data), &data)
	if err != nil {
		return db.GameResult{}, err
	}
	parsedState := PokerState{}
	err = json.Unmarshal([]byte(state.State), &parsedState)
	if err != nil {
		return db.GameResult{}, err
	}

	if data.Replace && len(data.ToReplace) != 5 {
		return db.GameResult{}, errors.New("Bad arguments")
	}

	if data.Replace {
		deck := make([]Card, len(g.InitialDeck))
		copy(deck, g.InitialDeck)

		for i := range 5 {
			handCard := parsedState.CardsInHand[i]
			if data.ToReplace[i] {
				continue
			}

			for ci := range len(deck) {
				if deck[ci].Number == handCard.Number && deck[ci].Suit == handCard.Suit {
					pickCard(uint64(ci), &deck)
					break
				}

			}
		}

		for i := range 5 {
			rng := randomNumbers[i]
			if data.ToReplace[i] {
				parsedState.CardsInHand[i] = pickCard(rng, &deck)
			}
		}
	}

	cardsInHandCopy := make([]Card, 5)
	copy(cardsInHandCopy, parsedState.CardsInHand)

	multiplier, outcome := determinePayout(cardsInHandCopy)
	profit := state.Amount.Mul(multiplier)

	returnState, _ := json.Marshal(parsedState)
	return db.GameResult{
		TotalProfit: profit,
		Outcomes:    []uint64{uint64(outcome)},
		Profits:     []decimal.Decimal{profit},
		NumGames:    1,
		Data:        string(returnState),
		Finished:    true,
	}, nil
}

func (*Poker) NumbersPerBet() uint64 {
	return 5
}

type By []Card

func (a By) Len() int           { return len(a) }
func (a By) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a By) Less(i, j int) bool { return a[i].Number < a[j].Number }

func determinePayout(sortedCards []Card) (decimal.Decimal, uint32) {
	sort.Sort(By(sortedCards))

	// Check 4 of a kind
	if sortedCards[1].Number == sortedCards[2].Number &&
		sortedCards[2].Number == sortedCards[3].Number {
		if sortedCards[1].Number == sortedCards[0].Number ||
			sortedCards[3].Number == sortedCards[4].Number {
			return decimal.New(30, 0), 7
		}
	}

	// Check full house -> 3 of a kind + pair
	if sortedCards[1].Number == sortedCards[0].Number &&
		sortedCards[4].Number == sortedCards[3].Number {
		if sortedCards[1].Number == sortedCards[2].Number ||
			sortedCards[3].Number == sortedCards[2].Number {
			return decimal.New(8, 0), 6
		}
	}

	// Check royal flush + straight flush + flush
	if sortedCards[0].Suit == sortedCards[1].Suit &&
		sortedCards[2].Suit == sortedCards[3].Suit &&
		sortedCards[0].Suit == sortedCards[4].Suit &&
		sortedCards[2].Suit == sortedCards[1].Suit {
		if sortedCards[0].Number == 1 && sortedCards[4].Number == 13 {
			if sortedCards[2].Number == sortedCards[3].Number-1 &&
				sortedCards[3].Number == sortedCards[4].Number-1 &&
				sortedCards[1].Number == sortedCards[2].Number-1 {
				return decimal.New(100, 0), 9
			}
		}
		if sortedCards[0].Number == 1 && sortedCards[1].Number == 2 {
			if sortedCards[0].Number == sortedCards[1].Number-1 &&
				sortedCards[2].Number == sortedCards[3].Number-1 &&
				sortedCards[3].Number == sortedCards[4].Number-1 &&
				sortedCards[1].Number == sortedCards[2].Number-1 {
				return decimal.New(50, 0), 8
			}
		}
		if sortedCards[0].Number == sortedCards[1].Number-1 &&
			sortedCards[2].Number == sortedCards[3].Number-1 &&
			sortedCards[3].Number == sortedCards[4].Number-1 &&
			sortedCards[1].Number == sortedCards[2].Number-1 {
			return decimal.New(50, 0), 8
		}
		return decimal.New(6, 0), 5
	}

	// Check straight
	if sortedCards[0].Number == 1 && sortedCards[1].Number == 2 {
		if sortedCards[0].Number == sortedCards[1].Number-1 &&
			sortedCards[2].Number == sortedCards[3].Number-1 &&
			sortedCards[3].Number == sortedCards[4].Number-1 &&
			sortedCards[1].Number == sortedCards[2].Number-1 {
			return decimal.New(5, 0), 4
		}
	}
	if sortedCards[0].Number == 1 && sortedCards[4].Number == 13 {
		if sortedCards[2].Number == sortedCards[3].Number-1 &&
			sortedCards[3].Number == sortedCards[4].Number-1 &&
			sortedCards[1].Number == sortedCards[2].Number-1 {
			return decimal.New(5, 0), 4
		}
	}
	if sortedCards[0].Number == sortedCards[1].Number-1 &&
		sortedCards[1].Number == sortedCards[2].Number-1 &&
		sortedCards[2].Number == sortedCards[3].Number-1 &&
		sortedCards[3].Number == sortedCards[4].Number-1 {
		return decimal.New(5, 0), 4
	}

	// Check three of a kind
	if sortedCards[0].Number == sortedCards[1].Number &&
		sortedCards[1].Number == sortedCards[2].Number {
		return decimal.New(3, 0), 3
	}
	if sortedCards[1].Number == sortedCards[2].Number &&
		sortedCards[2].Number == sortedCards[3].Number {
		return decimal.New(3, 0), 3
	}
	if sortedCards[2].Number == sortedCards[3].Number &&
		sortedCards[3].Number == sortedCards[4].Number {
		return decimal.New(3, 0), 3
	}

	// Check two pair
	if sortedCards[0].Number == sortedCards[1].Number {
		if sortedCards[2].Number == sortedCards[3].Number ||
			sortedCards[3].Number == sortedCards[4].Number {
			return decimal.New(2, 0), 2
		}
	}
	if sortedCards[1].Number == sortedCards[2].Number {
		if sortedCards[3].Number == sortedCards[4].Number {
			return decimal.New(2, 0), 2
		}
	}

	// Check one pair jacks or higher
	if sortedCards[0].Number == sortedCards[1].Number {
		if sortedCards[0].Number > 10 || sortedCards[0].Number == 1 {
			return decimal.New(1, 0), 1
		}
	}
	if sortedCards[1].Number == sortedCards[2].Number {
		if sortedCards[1].Number > 10 || sortedCards[1].Number == 1 {
			return decimal.New(1, 0), 1
		}
	}
	if sortedCards[2].Number == sortedCards[3].Number {
		if sortedCards[2].Number > 10 || sortedCards[2].Number == 1 {
			return decimal.New(1, 0), 1
		}
	}
	if sortedCards[3].Number == sortedCards[4].Number {
		if sortedCards[3].Number > 10 || sortedCards[3].Number == 1 {
			return decimal.New(1, 0), 1
		}
	}

	return decimal.Zero, 0
}
