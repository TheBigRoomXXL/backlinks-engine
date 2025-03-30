package main

import (
	"fmt"
	"log"
	"os"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/planner"
)

func main() {
	fmt.Println("Let's do it again.")
	p, err := planner.New()
	if err != nil {
		log.Fatal(err)
	}
	p.Run()
	err = p.Seed(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
