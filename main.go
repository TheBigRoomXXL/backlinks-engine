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

	p := planner.New()

	if seedsCSVPath != "" {
		err := p.Seed(seedsCSVPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	task := p.NextCrawl()
	if task == nil {
		log.Println("No tasks available")
		return
	}
	log.Println(task.Host)
	for _, p := range task.Pages {
		log.Printf("- %s\n", p.String())
	}
}
