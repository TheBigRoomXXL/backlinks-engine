package internal

import (
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

type Settings struct {
	DB_USER         string
	DB_PASSWORD     string
	DB_HOSTNAME     string
	DB_NAME         string
	DB_PORT         string
	LOG_PATH        string
	MAX_PARALLELISM int
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
	dbName, ok := os.LookupEnv("DB_NAME")
	if !ok {
		dbName = "backlinks"
	}
	dbPort, ok := os.LookupEnv("DB_PORT")
	if !ok {
		dbPort = "9000"
	}
	logPath, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		logPath = "errors.log"
	}

	var maxParallelismInt int
	collyMaxParallelism, ok := os.LookupEnv("MAX_PARALLELISM")
	if !ok {
		maxParallelismInt = runtime.NumCPU()
	} else {
		maxParallelismInt, err = strconv.Atoi(collyMaxParallelism)
		if err != nil {
			log.Fatal("Failed to parse MAX_PARALLELISM: %w", err)
		}
	}

	return &Settings{
		DB_USER:         dbUser,
		DB_PASSWORD:     dbPassword,
		DB_HOSTNAME:     dbHostname,
		DB_NAME:         dbName,
		DB_PORT:         dbPort,
		LOG_PATH:        logPath,
		MAX_PARALLELISM: maxParallelismInt,
	}

}
