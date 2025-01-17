package api

import (
	"greekkeepers.io/backend/communications"
	"greekkeepers.io/backend/config"
	"greekkeepers.io/backend/db"
	"greekkeepers.io/backend/engine"
)

type SharedController struct {
	Db                     *db.DB
	Env                    *config.Env
	Manager                *communications.Manager
	StatelessEngineChannel chan engine.Bet
	StatefulEngineChannel  chan engine.Bet
}
