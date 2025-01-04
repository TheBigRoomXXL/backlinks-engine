package vwww

import (
	"context"
	"expvar"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

func GenerateVWWW(ctx context.Context, nbPage int, nbSeed int, directoryPath string) error {
	// 1. Prepare file structure
	err := os.MkdirAll(directoryPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create vwww directory: %w", err)
	}

	if !strings.HasPrefix(directoryPath, "/") {
		directoryPath = directoryPath + "/"
	}

	// 2. Generate page
	ids := make([]string, nbPage)
	for i := 0; i < nbPage; i++ {
		ids[i] = uuid.NewString()
	}

	var wg sync.WaitGroup
	progress := expvar.NewInt("progress")
	semaphore := make(chan struct{}, 16)
	waitWG := make(chan struct{})

	go func() {
		defer close(semaphore)
		defer close(waitWG)

		for i := 0; i < nbPage; i++ {
			wg.Add(1)
			semaphore <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-semaphore }()

				// 2.1 Prepare a file for each page
				pageFile, err := os.Create(directoryPath + ids[i])
				if err != nil {
					log.Printf("failed to create page file %s: %s", ids[i], err)
				}
				defer pageFile.Close()

				// 2.2 Generate targets
				cyclicId := ids[(i+1)%nbPage] // Ensure all nodes are connected
				targets := randomSample(ids)
				fields := append([]string{cyclicId}, targets...)
				_, err = pageFile.WriteString(strings.Join(fields, "\n"))
				if err != nil {
					log.Printf("failed to wrtie to page file %s: %s", ids[i], err)
				}
				err = pageFile.Sync()
				if err != nil {
					log.Printf("failed to sync page file %s: %s", ids[i], err)
				}
				progress.Add(1)
			}()
		}
		wg.Wait()
		waitWG <- struct{}{}
	}()

OuterLoop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Canceled")
			return nil
		case <-time.After(time.Second):
			p := progress.Value()
			fmt.Printf("\rPages done: %d (%.1f %%)", p, 100*float32(p)/float32(nbPage))
		case <-waitWG:
			fmt.Println("waitWG")
			break OuterLoop
		}
	}

	// 3. Generate seeds
	seedFile, err := os.Create(directoryPath + "seeds")
	if err != nil {
		return fmt.Errorf("failed to create seeds file: %w", err)
	}
	defer seedFile.Close()
	for i := 0; i < nbSeed; i++ {
		seed := ids[rand.Intn(nbPage)]
		_, err = seedFile.WriteString(seed + "\n")
		if err != nil {
			return fmt.Errorf("failed to write to seed file: %w", err)
		}
	}
	err = seedFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync seed file: %w", err)
	}
	return nil
}
