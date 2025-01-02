package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/vwww"
)

func main() {
	ctx, shutdown := context.WithCancelCause(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-signalChan
		shutdown(errors.New(s.String()))
	}()

	err := cli(ctx)
	if err != nil {
		log.Fatal(err)
	}

}

func cli(ctx context.Context) error {
	if len(os.Args) < 2 {
		return errors.New("a command (crawl or vwww) is expected as argument")
	}

	cmd := os.Args[1]

	if cmd == "crawl" {
		// err := crawl.Crawl(os.Args[2:])
		// if err != nil {
		// 	return errors.New("crawl failed: ", err)
		// }
		go telemetry.MetricsReport(ctx)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				telemetry.ProcessedURL.Add(1)
			}
		}
	}

	if cmd == "vwww" {
		if len(os.Args) < 3 {
			return errors.New("vwww expect a subcommand (generate or serve) as argument")
		}

		subcmd := os.Args[2]
		if subcmd == "generate" {
			if len(os.Args) < 5 {
				return errors.New("generate expect 2 argument: nbPage and nbSeed")
			}
			nbPage, err := strconv.Atoi(os.Args[3])
			if err != nil {
				return fmt.Errorf("failed to parse nbPage: %w", err)
			}
			nbSeed, err := strconv.Atoi(os.Args[4])
			if err != nil {
				return fmt.Errorf("failed to parse nbSeed: %w", err)
			}
			t0 := time.Now()
			err = vwww.GenerateVWWW(nbPage, nbSeed, fmt.Sprintf("vwww/%d", nbPage))
			if err != nil {
				return fmt.Errorf("failed to generate vwww: %w", err)
			}
			fmt.Println("Time to generate:", time.Since(t0))
			return nil
		}

		if subcmd == "serve" {
			if len(os.Args) < 4 {
				return errors.New("serve expect a path to a dumped vwww")
			}
			err := vwww.NewVWWW(os.Args[3]).Serve()
			if err != nil {
				return fmt.Errorf("VWWW crashed: %w", err)
			}
		}

		return errors.New("invalid subcommand: generate or serve is expected")
	}

	return errors.New("invalid command: crawl or vwww is expected")
}
