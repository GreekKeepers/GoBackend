package main

import (
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"greekkeepers.io/backend/config"
	"greekkeepers.io/backend/db"
)

func main() {
	env := config.Env{}
	err := config.LoadEnv(&env)
	if err != nil {
		slog.Error("Error loading config", "err", err)
		return
	}

	DBUrl := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", env.DBHost, env.DBPort, env.DBUser, env.DBName, env.DBUserPwd)
	fmt.Println(DBUrl)
	DB, err := gorm.Open(postgres.Open(DBUrl), &gorm.Config{})
	if err != nil {
		slog.Error("Error connecting to db", "err", err)
		return
	} else {
		slog.Info("Connected to db")
	}

	db.RunMigrations(DB)
}
