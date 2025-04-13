package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/planner"
)

var seedsCSVPath string

func init() {
	flag.StringVar(&seedsCSVPath, "seeds", "", "Path to a CSV with a list of pages")
}

func main() {
	flag.Parse()

	p := planner.New()

	if seedsCSVPath != "" {
		err := p.Seed(seedsCSVPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	task := p.NextCrawl()
	fmt.Println(task.Host)
	for _, p := range task.Pages {
		fmt.Printf("- %s\n", p.String())
	}
}
