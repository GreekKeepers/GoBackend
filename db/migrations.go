package db

import (
	"gorm.io/gorm"
	"log"
)

func RunMigrations(db *gorm.DB) {
	// Create the custom enum type for the OAuthProvider
	err := db.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'oauth_provider') THEN CREATE TYPE oauth_provider AS ENUM ('local', 'google', 'facebook', 'twitter'); END IF; END $$;").Error
	if err != nil {
		log.Fatalf("failed to create enum type: %v", err)
	}

	// Automatically migrate the schemas
	err = db.AutoMigrate(&User{}, &RefreshToken{}, &Coin{}, &Amount{}, &Game{}, &UserSeed{}, &ServerSeed{}, &Bet{}, &Payout{}, &GameState{})
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Ensure unique indexes
	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS amount_unique_idx ON amounts (user_id, coin_id);").Error
	if err != nil {
		log.Fatalf("failed to create unique index for amounts: %v", err)
	}

	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS user_seed_unique_idx ON user_seeds (user_id, user_seed);").Error
	if err != nil {
		log.Fatalf("failed to create unique index for user seeds: %v", err)
	}

	err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS state_unique_idx ON game_states (game_id, user_id, coin_id, user_seed_id, server_seed_id);").Error
	if err != nil {
		log.Fatalf("failed to create unique index for game states: %v", err)
	}

	err = db.Exec("INSERT INTO Coins(name, price) VALUES ('DraxBonus',1000);").Error
	if err != nil {
		log.Printf("failed to create unique index for game states: %v", err)
	}
	err = db.Exec("INSERT INTO Coins(name, price) VALUES ('Drax',10);").Error
	if err != nil {
		log.Printf("failed to create unique index for game states: %v", err)
	}
}
