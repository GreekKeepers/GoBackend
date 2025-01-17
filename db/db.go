package db

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"greekkeepers.io/backend/responses"
)

type DB struct {
	*gorm.DB
}

func (db *DB) DecreaseBalance(userId uint, coinId uint, amount decimal.Decimal) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		balance := Amount{}
		err := tx.Where("coin_id = ? AND user_id = ?", coinId, userId).First(&balance).Error
		if err != nil {
			return err
		}

		if amount.GreaterThan(balance.Amount) {
			return errors.New("Amount is greater, than balance")
		}

		if err := tx.Where("user_id=? AND coin_id=?", userId, coinId).Update("amount", balance.Amount.Sub(amount)).Error; err != nil {
			return err
		}

		return nil
	})

	return err
}

func (db *DB) RemoveGameState(gameId uint, userId uint, coinId uint) error {
	gameState := GameState{}
	err := db.Where("game_id=? AND user_id=? AND coin_id=?", gameId, userId, coinId).Delete(&gameState).Error

	return err
}

func (db *DB) InsertGameState(gameId uint, userId uint, uuid string, coinId uint, data string, state string, amount decimal.Decimal, userSeedId uint, serverSeedId uint, timestamp time.Time) error {
	gameState := GameState{
		Timestamp:    timestamp,
		Amount:       amount,
		BetInfo:      data,
		State:        state,
		UUID:         uuid,
		GameID:       gameId,
		UserID:       userId,
		CoinID:       coinId,
		UserSeedID:   userSeedId,
		ServerSeedID: serverSeedId,
	}
	err := db.Create(&gameState).Error

	return err
}

func (db *DB) GetGameState(gameId uint, userId uint, coinId uint) (GameState, error) {
	gameState := GameState{}
	err := db.Where("game_id=? AND user_id=? AND coin_id=?", gameId, userId, coinId).First(&gameState).Error

	return gameState, err
}

func (db *DB) UpdateGameState(gameId uint, userId uint, coinId uint, state string) error {
	err := db.Where("game_id=? AND user_id=? AND coin_id=?", gameId, userId, coinId).Update("state", &state).Error

	return err
}

func (db *DB) IncreaseBalance(userId uint, coinId uint, amount decimal.Decimal) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		balance := Amount{}
		err := tx.Where("coin_id = ? AND user_id = ?", coinId, userId).First(&balance).Error
		if err != nil {
			return err
		}

		if err := tx.Where("user_id=? AND coin_id=?", userId, coinId).Update("amount", balance.Amount.Add(amount)).Error; err != nil {
			return err
		}

		return nil
	})
	return err
}

func (db *DB) SubIncBalance(userId uint, coinId uint, subAmount decimal.Decimal, addAmount decimal.Decimal) error {
	err := db.Transaction(func(tx *gorm.DB) error {

		balance := Amount{}
		err := tx.Where("coin_id = ? AND user_id = ?", coinId, userId).First(&balance).Error
		if err != nil {
			return err
		}

		if subAmount.GreaterThan(balance.Amount) {
			return errors.New("Amount is greater, than balance")
		}

		if err := tx.Model(&Amount{}).Where("user_id=? AND coin_id=?", userId, coinId).Update("amount", balance.Amount.Sub(subAmount)).Error; err != nil {
			return err
		}

		err = tx.Where("coin_id = ? AND user_id = ?", coinId, userId).First(&balance).Error
		if err != nil {
			return err
		}

		if err := tx.Model(&Amount{}).Where("user_id=? AND coin_id=?", userId, coinId).Update("amount", balance.Amount.Add(addAmount)).Error; err != nil {
			return err
		}

		return nil
	})

	return err
}

func (db *DB) FetchLeaderboardVolume(timeBoundaries string) ([]responses.Leaderboard, error) {
	result := make([]responses.Leaderboard, 20)
	items := int64(0)
	switch timeBoundaries {
	case "daily":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM((bets.amount*bets.num_games)/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 day'
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "weekly":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM((bets.amount*bets.num_games)/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 week'
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "monthly":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM((bets.amount*bets.num_games)/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 month'
                        GROUP BY bet.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "all":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM((bets.amount*bets.num_games)/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        GROUP BY bet.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	default:
		return nil, errors.New("Unknown Time boundary")
	}

	return result[:items], nil
}

func (db *DB) FetchLeaderboardProfit(timeBoundaries string) ([]responses.Leaderboard, error) {
	result := make([]responses.Leaderboard, 20)
	items := int64(0)
	switch timeBoundaries {
	case "daily":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM(bets.profit/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 day'
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT ?`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "weekly":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM(bets.profit/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 week'
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "monthly":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM(bets.profit/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        WHERE bets.timestamp > now() - interval '1 month'
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	case "all":
		res := db.Raw(`SELECT bets.user_id, bets.total, Users.username FROM (
                        SELECT 
                            bets.user_id, 
                            SUM(bets.profit/Coins.price) as total
                        FROM bets
                        INNER JOIN Coins ON Coins.id=bets.coin_id
                        GROUP BY bets.user_id) as bets
                INNER JOIN Users ON Users.id=bets.user_id
                ORDER BY total DESC
                LIMIT $1`, 20).Scan(&result)
		items = res.RowsAffected
		if res.Error != nil {
			return nil, res.Error
		}
		break
	default:
		return nil, errors.New("Unknown Time boundary")
	}

	return result[:items], nil
}
