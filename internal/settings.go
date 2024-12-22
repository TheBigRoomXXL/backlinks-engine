package internal

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Settings struct {
	DB_USER     string
	DB_PASSWORD string
	LOG_PATH    string
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
	logPath, ok := os.LookupEnv("LOG_PATH")
	if !ok {
		logPath = "errors.log"
	}

	return &Settings{
		DB_USER:     dbUser,
		DB_PASSWORD: dbPassword,
		LOG_PATH:    logPath,
	}

}

func NewDatabase(s *Settings) (neo4j.DriverWithContext, error) {
	uri := "neo4j://localhost:7687" // TODO: add to settings

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(s.DB_USER, s.DB_PASSWORD, ""))
	if err != nil {
		return nil, err
	}

	// Test the connection by verifying the authentication
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify connection: %w", err)
	}

	return driver, nil
}
