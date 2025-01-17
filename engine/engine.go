package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/blake2b"
	"greekkeepers.io/backend/communications"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/games"
	"greekkeepers.io/backend/requests"
	"greekkeepers.io/backend/responses"
)

func ParseStatelessGame(
	gameName string,
	params string,
) (games.StatelessGameEngine, error) {
	switch gameName {
	case "CoinFlip":
		var game games.CoinFlip
		err := json.Unmarshal([]byte(params), &game)
		if err != nil {
			slog.Error("Error parsing CoinFlip", "err", err)
			return nil, errors.New("Error parsing CoinFlip")
		}
		return &game, nil
	case "Plinko":
		var game games.Plinko
		err := json.Unmarshal([]byte(params), &game)
		if err != nil {
			slog.Error("Error parsing Plinko", "err", err)
			return nil, errors.New("Error parsing Plinko")
		}
		return &game, nil
	}
	return nil, nil
}

func ParseStatefulGame(
	gameName string,
	params string,
) (games.StatefulGameEngine, error) {
	return nil, nil
}

func GenerateRandomNumbers(
	clientSeed string,
	serverSeed string,
	timestamp uint64,
	amount uint64,
) []uint64 {
	postfix := fmt.Sprintf("%d%s%s", timestamp, clientSeed, serverSeed)

	result := make([]uint64, amount)
	for i := range amount {
		hash := blake2b.Sum256([]byte(fmt.Sprintf("%d", i) + postfix))
		number := uint64(hash[0])<<56 | uint64(hash[1])<<48 | uint64(hash[2])<<40 | uint64(hash[3])<<32 | uint64(hash[4])<<24 | uint64(hash[5])<<16 | uint64(hash[6])<<8 | uint64(hash[7])

		result[i] = number
	}

	return result
}

type Bet struct {
	IsContinue bool
	Bet        interface{}
}

type StatelessEngine struct {
	BetReceiver        chan Bet
	StatefulBetChannel chan Bet
	Games              map[uint]games.StatelessGameEngine
	Manager            *communications.Manager
	Db                 *db.DB
}

func NewStatelessEngine(
	BetReceiver chan Bet,
	StatefulBetChannel chan Bet,
	Manager *communications.Manager,
	Db *db.DB,
) StatelessEngine {
	var gamesRaw []db.Game

	err := Db.Find(&gamesRaw).Error
	if err != nil {
		slog.Error("Error retrieving games", "err", err)
		panic("Error retrieving games")
	}

	games := make(map[uint]games.StatelessGameEngine)
	for _, game := range gamesRaw {
		gameParsed, err := ParseStatelessGame(game.Name, game.Parameters)
		if err != nil {
			panic("Error parsing game")
		}
		games[game.ID] = gameParsed
	}

	return StatelessEngine{
		BetReceiver:        BetReceiver,
		StatefulBetChannel: StatefulBetChannel,
		Games:              games,
		Db:                 Db,
		Manager:            Manager,
	}
}

func (e *StatelessEngine) Run() {
	slog.Info("Starting stateless engine")
	for {
		origBet, ok := <-e.BetReceiver
		if !ok {
			slog.Error("Bet Receiver channel is closed")
			break
		}
		slog.Info("Received bet", "bet", origBet)

		if origBet.IsContinue {
			e.StatefulBetChannel <- origBet
		}

		bet := origBet.Bet.(requests.Bet)
		if bet.NumGames > 100 {
			continue
		}

		engine, ok := e.Games[bet.GameID]
		if !ok {
			slog.Warn("GameID wasn't found", "bet", bet)
			e.StatefulBetChannel <- origBet
			continue
		}

		coin := db.Coin{}
		err := e.Db.Where("id = ?", bet.CoinID).First(&coin).Error
		if err != nil {
			slog.Error("Error getting coing", "bet", bet, "err", err)
			continue
		}

		fullBetAmount := bet.Amount.Mul(decimal.NewFromUint64(bet.NumGames))
		fullBetAmountInUsd := fullBetAmount.Div(coin.Price)

		if fullBetAmountInUsd.GreaterThan(decimal.New(50, 0)) {
			continue
		}

		balance := db.Amount{}
		err = e.Db.Where("coin_id = ? AND user_id = ?", coin.ID, bet.UserID).First(&balance).Error
		if err != nil {
			slog.Error("Error getting user balance", "bet", bet, "err", err)
			continue
		}

		if fullBetAmount.GreaterThan(balance.Amount) {
			continue
		}

		userSeed := &db.UserSeed{}
		err = e.Db.Where("user_id = ?", bet.UserID).Order("created_at DESC").First(userSeed).Error
		if err != nil {
			slog.Error("Error getting user seed", "bet", bet, "err", err)
			continue
		}

		serverSeed := &db.ServerSeed{}
		if err := e.Db.Where("user_id=? AND revealed=FALSE", bet.UserID).First(&serverSeed).Error; err != nil {
			slog.Error("Failed adding a seed", "bet", bet, "err", err)
			continue
		}

		timeNow := time.Now()
		timestamp := uint64(timeNow.Unix())
		randomNumbers := GenerateRandomNumbers(
			userSeed.UserSeed,
			serverSeed.ServerSeed,
			timestamp,
			engine.NumbersPerBet()*bet.NumGames,
		)

		gameResult, err := engine.Play(bet, randomNumbers)
		if err != nil {
			slog.Warn("Failed to proccess bet", "bet", bet, "err", err)
			continue
		}

		totalSpent := bet.Amount.Mul(decimal.NewFromInt32(int32(gameResult.NumGames)))

		err = e.Db.SubIncBalance(bet.UserID, bet.CoinID, totalSpent, gameResult.TotalProfit)
		if err != nil {
			slog.Error("Error updating balance", "bet", bet, "err", err)
			continue
		}

		outcomes, err := json.Marshal(gameResult.Outcomes)
		if err != nil {
			slog.Error("Error marshaling outcomes", "gameResult", gameResult, "err", err)
			continue
		}

		profits, err := json.Marshal(gameResult.Profits)
		if err != nil {
			slog.Error("Error marshaling profits", "gameResult", gameResult, "err", err)
			continue
		}

		dbBet := db.Bet{
			Timestamp:    timeNow,
			Amount:       fullBetAmount,
			Profit:       gameResult.TotalProfit,
			NumGames:     int(gameResult.NumGames),
			Outcomes:     string(outcomes[:]),
			Profits:      string(profits[:]),
			BetInfo:      bet.Data,
			UUID:         bet.UUID,
			GameID:       bet.GameID,
			UserID:       bet.UserID,
			CoinID:       bet.CoinID,
			UserSeedID:   userSeed.ID,
			ServerSeedID: serverSeed.ID,
		}
		err = e.Db.Create(&dbBet).Error
		if err != nil {
			slog.Error("Error placing bet", "bet", bet, "dbbet", dbBet, "err", err)
			continue
		}

		var userFull db.User
		if err := e.Db.Where("id = ?", bet.UserID).First(&userFull).Error; err != nil {
			slog.Error("User not found", "userId", bet.UserID)
			continue
		}

		constructedBet := responses.Bet{
			ID:           0,
			Timestamp:    timeNow,
			Amount:       fullBetAmount,
			Profit:       gameResult.TotalProfit,
			NumGames:     int(gameResult.NumGames),
			Outcomes:     string(outcomes[:]),
			Profits:      string(profits[:]),
			BetInfo:      bet.Data,
			UUID:         bet.UUID,
			GameID:       bet.GameID,
			UserID:       bet.UserID,
			Username:     userFull.Username,
			CoinID:       bet.CoinID,
			UserSeedID:   userSeed.ID,
			ServerSeedID: serverSeed.ID,
		}

		e.Manager.ManagerReceiver <- communications.ManagerEvent{
			Type: communications.PropagateBet,
			Body: constructedBet,
		}

	}
}

type StatefulEngine struct {
	StatefulBetChannel chan Bet
	Games              map[uint]games.StatefulGameEngine
	Manager            *communications.Manager
	Db                 *db.DB
}

func NewStatefulEngine(
	BetReceiver chan Bet,
	Manager *communications.Manager,
	Db *db.DB,
) StatefulEngine {
	var gamesRaw []db.Game

	err := Db.Find(&gamesRaw).Error
	if err != nil {
		slog.Error("Error retrieving games", "err", err)
		panic("Error retrieving games")
	}

	games := make(map[uint]games.StatefulGameEngine)
	for _, game := range gamesRaw {
		gameParsed, err := ParseStatefulGame(game.Name, game.Parameters)
		if err != nil {
			panic("Error parsing game")
		}
		games[game.ID] = gameParsed
	}

	return StatefulEngine{
		StatefulBetChannel: BetReceiver,
		Games:              games,
		Manager:            Manager,
		Db:                 Db,
	}
}

func (e *StatefulEngine) Run() {
	slog.Info("Starting stateful engine")
	for {
		origBet, ok := <-e.StatefulBetChannel
		if !ok {
			slog.Error("Stateful Receiver channel is closed")
			break
		}

		if !origBet.IsContinue {
			bet := origBet.Bet.(requests.Bet)
			if bet.NumGames > 100 {
				continue
			}

			engine, ok := e.Games[bet.GameID]
			if !ok {
				slog.Warn("Stateful GameID wasn't found", "bet", bet)
				continue
			}
			coin := db.Coin{}
			err := e.Db.Where("id = ?", bet.CoinID).First(&coin).Error
			if err != nil {
				slog.Error("Error getting coing", "bet", bet, "err", err)
				continue
			}
			fullBetAmount := bet.Amount.Mul(decimal.NewFromUint64(bet.NumGames))
			fullBetAmountInUsd := fullBetAmount.Div(coin.Price)
			if fullBetAmountInUsd.GreaterThan(decimal.New(50, 0)) {
				continue
			}
			balance := db.Amount{}
			err = e.Db.Where("coin_id = ? AND user_id = ?", coin.ID, bet.UserID).First(&balance).Error
			if err != nil {
				slog.Error("Error getting user balance", "bet", bet, "err", err)
				continue
			}
			if fullBetAmount.GreaterThan(balance.Amount) {
				continue
			}
			userSeed := &db.UserSeed{}
			err = e.Db.Where("user_id = ?", bet.UserID).Order("created_at DESC").First(userSeed).Error
			if err != nil {
				slog.Error("Error getting user seed", "bet", bet, "err", err)
				continue
			}
			serverSeed := &db.ServerSeed{}
			if err := e.Db.Where("user_id=? AND revealed=FALSE", bet.UserID).First(&serverSeed).Error; err != nil {
				slog.Error("Failed adding a seed", "bet", bet, "err", err)
				continue
			}
			timeNow := time.Now()
			timestamp := uint64(timeNow.Unix())
			randomNumbers := GenerateRandomNumbers(
				userSeed.UserSeed,
				serverSeed.ServerSeed,
				timestamp,
				engine.NumbersPerBet(),
			)
			gameResult, err := engine.Play(bet, randomNumbers)
			if err != nil {
				slog.Warn("Failed to proccess bet", "bet", bet, "err", err)
				continue
			}

			err = e.Db.DecreaseBalance(bet.UserID, bet.CoinID, bet.Amount)
			if err != nil {
				slog.Error("Error updating balance", "bet", bet, "err", err)
				continue
			}
			if gameResult.Finished {
				// Game finished
				if !gameResult.TotalProfit.IsZero() {
					err = e.Db.IncreaseBalance(bet.UserID, bet.CoinID, gameResult.TotalProfit)
					if err != nil {
						slog.Error("Error updating balance", "bet", bet, "err", err)
						continue
					}
				}

				outcomes, err := json.Marshal(gameResult.Outcomes)
				if err != nil {
					slog.Error("Error marshaling outcomes", "gameResult", gameResult, "err", err)
					continue
				}

				profits, err := json.Marshal(gameResult.Profits)
				if err != nil {
					slog.Error("Error marshaling profits", "gameResult", gameResult, "err", err)
					continue
				}

				dbBet := db.Bet{
					Timestamp:    timeNow,
					Amount:       fullBetAmount,
					Profit:       gameResult.TotalProfit,
					NumGames:     int(gameResult.NumGames),
					Outcomes:     string(outcomes[:]),
					Profits:      string(profits[:]),
					BetInfo:      bet.Data,
					UUID:         bet.UUID,
					GameID:       bet.GameID,
					UserID:       bet.UserID,
					CoinID:       bet.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}
				err = e.Db.Create(&dbBet).Error
				if err != nil {
					slog.Error("Error placing bet", "bet", bet, "dbbet", dbBet, "err", err)
					continue
				}

				if gameResult.NumGames > 1 {
					err := e.Db.RemoveGameState(bet.GameID, bet.UserID, bet.CoinID)
					if err != nil {
						slog.Error("Error removing game state", "err", err)
						continue
					}
				}

				var userFull db.User
				if err := e.Db.Where("id = ?", bet.UserID).First(&userFull).Error; err != nil {
					slog.Error("User not found", "userId", bet.UserID)
					continue
				}

				constructedBet := responses.Bet{
					ID:           0,
					Timestamp:    timeNow,
					Amount:       fullBetAmount,
					Profit:       gameResult.TotalProfit,
					NumGames:     int(gameResult.NumGames),
					Outcomes:     string(outcomes[:]),
					Profits:      string(profits[:]),
					BetInfo:      bet.Data,
					UUID:         bet.UUID,
					GameID:       bet.GameID,
					UserID:       bet.UserID,
					Username:     userFull.Username,
					CoinID:       bet.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}
				e.Manager.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.PropagateBet,
					Body: constructedBet,
				}
			} else {
				// game state changed
				err := e.Db.InsertGameState(
					bet.GameID,
					bet.UserID,
					bet.UUID,
					bet.CoinID,
					bet.Data,
					gameResult.Data,
					bet.Amount,
					userSeed.ID,
					serverSeed.ID,
					timeNow,
				)
				if err != nil {
					slog.Error("Error inserting game state", "err", err)
					continue
				}

				state := db.GameState{
					ID:           0,
					Timestamp:    timeNow,
					Amount:       fullBetAmount,
					BetInfo:      bet.Data,
					State:        gameResult.Data,
					UUID:         bet.UUID,
					GameID:       bet.GameID,
					UserID:       bet.UserID,
					CoinID:       bet.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}

				e.Manager.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.PropagateState,
					Body: state,
				}
			}
		} else {
			continueGame := origBet.Bet.(requests.ContinueGame)

			engine, ok := e.Games[continueGame.GameID]
			if !ok {
				slog.Warn("Stateful GameID wasn't found", "bet", continueGame)
				continue
			}

			state, err := e.Db.GetGameState(continueGame.GameID, continueGame.UserID, continueGame.CoinID)
			if err != nil {
				slog.Error("Error getting game state")
				continue
			}

			userSeed := &db.UserSeed{}
			err = e.Db.Where("user_id = ?", continueGame.UserID).Order("created_at DESC").First(userSeed).Error
			if err != nil {
				slog.Error("Error getting user seed", "bet", continueGame, "err", err)
				continue
			}
			serverSeed := &db.ServerSeed{}
			if err := e.Db.Where("user_id=? AND revealed=FALSE", continueGame.UserID).First(&serverSeed).Error; err != nil {
				slog.Error("Failed adding a seed", "bet", continueGame, "err", err)
				continue
			}

			timeNow := time.Now()
			timestamp := uint64(timeNow.Unix())
			randomNumbers := GenerateRandomNumbers(
				userSeed.UserSeed,
				serverSeed.ServerSeed,
				timestamp,
				engine.NumbersPerBet(),
			)
			gameResult, err := engine.ContinuePlaying(state, continueGame, randomNumbers)
			if err != nil {
				slog.Warn("Failed to proccess bet", "bet", continueGame, "state", state, "err", err)
				continue
			}

			if gameResult.Finished {
				// Game finished
				err := e.Db.RemoveGameState(continueGame.GameID, continueGame.UserID, continueGame.CoinID)
				if err != nil {
					slog.Error("Error removing game state", "err", err)
					continue
				}

				if !gameResult.TotalProfit.IsZero() {
					err = e.Db.IncreaseBalance(continueGame.UserID, continueGame.CoinID, gameResult.TotalProfit)
					if err != nil {
						slog.Error("Error updating balance", "bet", continueGame, "err", err)
						continue
					}
				}

				outcomes, err := json.Marshal(gameResult.Outcomes)
				if err != nil {
					slog.Error("Error marshaling outcomes", "gameResult", gameResult, "err", err)
					continue
				}

				profits, err := json.Marshal(gameResult.Profits)
				if err != nil {
					slog.Error("Error marshaling profits", "gameResult", gameResult, "err", err)
					continue
				}

				dbBet := db.Bet{
					Timestamp:    timeNow,
					Amount:       state.Amount,
					Profit:       gameResult.TotalProfit,
					NumGames:     int(gameResult.NumGames),
					Outcomes:     string(outcomes[:]),
					Profits:      string(profits[:]),
					BetInfo:      continueGame.Data,
					UUID:         continueGame.UUID,
					GameID:       continueGame.GameID,
					UserID:       continueGame.UserID,
					CoinID:       continueGame.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}
				err = e.Db.Create(&dbBet).Error
				if err != nil {
					slog.Error("Error placing bet", "bet", continueGame, "dbbet", dbBet, "err", err)
					continue
				}

				var userFull db.User
				if err := e.Db.Where("id = ?", continueGame.UserID).First(&userFull).Error; err != nil {
					slog.Error("User not found", "userId", continueGame.UserID)
					continue
				}

				constructedBet := responses.Bet{
					ID:           0,
					Timestamp:    timeNow,
					Amount:       state.Amount,
					Profit:       gameResult.TotalProfit,
					NumGames:     int(gameResult.NumGames),
					Outcomes:     string(outcomes[:]),
					Profits:      string(profits[:]),
					BetInfo:      continueGame.Data,
					UUID:         continueGame.UUID,
					GameID:       continueGame.GameID,
					UserID:       continueGame.UserID,
					Username:     userFull.Username,
					CoinID:       continueGame.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}
				e.Manager.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.PropagateBet,
					Body: constructedBet,
				}
			} else {
				// game state changed
				err := e.Db.InsertGameState(
					continueGame.GameID,
					continueGame.UserID,
					continueGame.UUID,
					continueGame.CoinID,
					continueGame.Data,
					gameResult.Data,
					state.Amount,
					userSeed.ID,
					serverSeed.ID,
					timeNow,
				)
				if err != nil {
					slog.Error("Error inserting game state", "err", err)
					continue
				}

				state := db.GameState{
					ID:           0,
					Timestamp:    timeNow,
					Amount:       state.Amount,
					BetInfo:      continueGame.Data,
					State:        gameResult.Data,
					UUID:         continueGame.UUID,
					GameID:       continueGame.GameID,
					UserID:       continueGame.UserID,
					CoinID:       continueGame.CoinID,
					UserSeedID:   userSeed.ID,
					ServerSeedID: serverSeed.ID,
				}

				e.Manager.ManagerReceiver <- communications.ManagerEvent{
					Type: communications.PropagateState,
					Body: state,
				}
			}
		}

	}
}
