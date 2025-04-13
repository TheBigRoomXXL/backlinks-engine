package main

import (
	"flag"
	"log"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/planner"
)

var seedsCSVPath string

func init() {
	flag.StringVar(&seedsCSVPath, "seeds", "", "Path to a CSV with a list of pages")
}

func main() {
	flag.Parse()

	p, err := planner.New()
	if err != nil {
		log.Fatal(err)
	}

	if seedsCSVPath != "" {
		err = p.Seed(seedsCSVPath)
		if err != nil {
			log.Fatal(err)
		}
	}
}
