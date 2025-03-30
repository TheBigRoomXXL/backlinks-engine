package planner

import (
	"errors"
	"os"
	"sync"
)

type Settings struct {
	DB_USER     string
	DB_PASSWORD string
	DB_HOSTNAME string
	DB_PORT     string
	DB_NAME     string
	DB_TLS      bool
}

var (
	settings *Settings
	initOnce sync.Once
	initErr  error
)

// Initialize a new settings object from environnment variable. dotenv file is not supported
func newSettings() (*Settings, error) {
	initOnce.Do(initSettings)
	return settings, initErr
}

func initSettings() {
	dbUser, ok := os.LookupEnv("DB_USER")
	if !ok {
		dbUser = "backlinks-engine"
	}

	dbPassword, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		initErr = errors.New("environnment $DB_PASSWORD is not set, defaulting to \"\"")
		return
	}

	dbHostname, ok := os.LookupEnv("DB_HOSTNAME")
	if !ok {
		dbHostname = "localhost"
	}

	dbPort, ok := os.LookupEnv("DB_PORT")
	if !ok {
		dbPort = "9000"
	}

	dbName, ok := os.LookupEnv("DB_NAME")
	if !ok {
		dbName = "backlinks"
	}

	settings = &Settings{
		DB_USER:     dbUser,
		DB_PASSWORD: dbPassword,
		DB_HOSTNAME: dbHostname,
		DB_PORT:     dbPort,
		DB_NAME:     dbName,
	}
}
