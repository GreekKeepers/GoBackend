package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Env struct {
	// server
	ServerHost string `envconfig:"SERVER_HOST"`
	ServerPort string `envconfig:"SERVER_PORT"`
	// database
	DBHost    string `envconfig:"DB_HOST"`
	DBName    string `envconfig:"DB_NAME"`
	DBPort    string `envconfig:"DB_PORT"`
	DBUser    string `envconfig:"DB_USER"`
	DBUserPwd string `envconfig:"DB_USER_PWD"`

	PageSize             uint64 `envconfig:"PAGE_SIZE"`
	PasswordSalt         string `envconfig:"PASSWORD_SALT"`
	RefreshTokenValidity uint64 `envconfig:"REFRESH_TOKEN_VALIDITY"`

	ENGINES uint16 `envconfig:"ENGINES"`
}

func LoadEnv(cfg *Env) error {
	err := godotenv.Load(".env")
	if err != nil {
		return err
	}
	err = envconfig.Process("", cfg)
	if err != nil {
		return err
	}

	return nil
}
