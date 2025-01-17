package main

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"greekkeepers.io/backend/api"
	"greekkeepers.io/backend/communications"
	"greekkeepers.io/backend/config"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/engine"
)

func main() {
	env := config.Env{}
	err := config.LoadEnv(&env)
	if err != nil {
		slog.Error("Error loading config", "err", err)
		return
	}
	router := gin.Default()

	DBUrl := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", env.DBHost, env.DBPort, env.DBUser, env.DBName, env.DBUserPwd)
	DB, err := gorm.Open(postgres.Open(DBUrl), &gorm.Config{})
	if err != nil {
		slog.Error("Error connecting to db", "err", err)
		return
	} else {
		slog.Info("Connected to db")
	}

	statefulBetChannel := make(chan engine.Bet)
	statelessBetChannel := make(chan engine.Bet)

	communications.New(DB)
	go communications.ManagerPub.Run()
	sCtrl := api.SharedController{Db: &db.DB{DB}, Env: &env, Manager: communications.ManagerPub, StatelessEngineChannel: statelessBetChannel}

	stateless := engine.NewStatelessEngine(statelessBetChannel, statefulBetChannel, communications.ManagerPub, &db.DB{DB})
	stateful := engine.NewStatefulEngine(statefulBetChannel, communications.ManagerPub, &db.DB{DB})

	go stateless.Run()
	go stateful.Run()

	router.Use(api.CORSMiddleware())

	api.AuthEndpoints(&sCtrl, router)
	api.UserEndpoints(&sCtrl, router)
	api.GameEndpoints(&sCtrl, router)
	api.GeneralEndpoints(&sCtrl, router)
	api.BetsEndpoints(&sCtrl, router)
	api.CoinEndpoints(&sCtrl, router)
	api.ReferalEndpoints(&sCtrl, router)
	router.Run(fmt.Sprintf("%s:%s", env.ServerHost, env.ServerPort))

}
