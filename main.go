package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Settings struct {
	NEO4J_USER     string
	NEO4J_PASSWORD string
}

func newSettings() *Settings {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return &Settings{
		NEO4J_USER:     os.Getenv("NEO4J_USER"),
		NEO4J_PASSWORD: os.Getenv("NEO4J_PASSWORD"),
	}

}

func newNeo4j(s *Settings) (neo4j.DriverWithContext, error) {
	uri := "neo4j://localhost:7687" // TODO: add to settings

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(s.NEO4J_USER, s.NEO4J_PASSWORD, ""))
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

func Main() {
	s := newSettings()

	db, err := newNeo4j(s)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	Crawl(s, db)
}
