package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
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

func main() {

	s := newSettings()

	db, err := newDatabase(s)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	if len(os.Args) < 2 {
		log.Fatal("A command (crawl or vwww) is expected as argument")
	}

	cmd := os.Args[1]
	if cmd != "crawl" && cmd != "vwww" {
		log.Fatal("Invalid command: crawl or vwww is expected")
	}
	if cmd == "crawl" {
		Crawl(s, db, os.Args[2:])
	}
	if cmd == "vwww" {
		vwww := NewVirtualWorldWideWeb(1_000)
		ServeVWWW(vwww)
	}
}
