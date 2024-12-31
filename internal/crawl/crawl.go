package crawl

import (
	"fmt"

	planer "github.com/TheBigRoomXXL/backlinks-engine/internal/planner"
)

func Crawl(seeds []string) error {
	fmt.Println("seeds are:", seeds)
	planner, err := planer.New()
	if err != nil {
		return fmt.Errorf("failed to create planner: %w", err)
	}

	err = planner.Seed(seeds)
	if err != nil {
		return fmt.Errorf("failed to import seeds: %w", err)
	}

	for i := 0; i < 10; i++ {
		fmt.Print(i, " -> ")
		pages, err := planner.Next()
		if err != nil {
			return err
		}
		fmt.Println(pages)
	}

	return nil
}
