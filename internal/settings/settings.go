package settings

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Settings struct {
	DB_USER     string
	DB_PASSWORD string
	DB_HOSTNAME string
	DB_PORT     string
	DB_NAME     string
	DB_OPTIONS  string
	LOG_PATH    string
	PPROF_PORT  string
}

func NewSettings() *Settings {
	err := godotenv.Load(".env")
	if err != nil && err.Error() != "open .env: no such file or directory" {
		log.Fatal("Error loading .env file")
	}

	dbUser, ok := os.LookupEnv("DB_USER")
	if !ok {
		log.Fatal("$DB_USER must be set.")
	}
	dbPassword, ok := os.LookupEnv("DB_PASSWORD")
	if !ok {
		log.Fatal("$DB_PASSWORD must be set")
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

	dbOptions, ok := os.LookupEnv("DB_OPTIONS")
	if !ok {
		dbName = ""
	}

	logPath, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		logPath = "errors.log"
	}

	pprofPort, ok := os.LookupEnv("PPROF_PORT")
	if !ok {
		pprofPort = "8081"
	}

	return &Settings{
		DB_USER:     dbUser,
		DB_PASSWORD: dbPassword,
		DB_HOSTNAME: dbHostname,
		DB_PORT:     dbPort,
		DB_NAME:     dbName,
		DB_OPTIONS:  dbOptions,
		LOG_PATH:    logPath,
		PPROF_PORT:  pprofPort,
	}

}