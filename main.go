package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/crawler"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/queue"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/telemetry"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/vwww"
)

func main() {
	ctx, shutdown := context.WithCancelCause(context.Background())
	defer func() {
		shutdown(nil)
		// We need to let some time pass so that goroutine that are waiting for this
		// cancelation have the time to react. Better techniques exists but are not worth
		// the hassle in this case
		time.Sleep(time.Millisecond * 10)
	}()

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
		go telemetry.MetricsReport(ctx)

		s, err := settings.New()
		if err != nil {
			return fmt.Errorf("faield to get setttings: %w", err)
		}
		fetcher := client.NewCrawlClient(
			ctx, http.DefaultTransport, s.HTTP_RATE_LIMIT, s.HTTP_MAX_RETRY, s.HTTP_TIMEOUT,
		)
		queue := queue.NewFIFOQueue()
		crawler, err := crawler.NewCrawler(ctx, queue, fetcher)

		if err != nil {
			return fmt.Errorf("failed to initialize crawler: %w", err)
		}
		for _, seed := range os.Args[2:] {
			url, err := url.Parse(seed)
			if err != nil {
				return fmt.Errorf("failed to parsed seed: %w", err)
			}
			url, err = commons.NormalizeUrl(url)
			if err != nil {
				return fmt.Errorf("failed to normalize seed: %w", err)
			}
			fmt.Println(url)
			crawler.AddUrl(url)
		}

		return crawler.Run()
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
			err = vwww.GenerateVWWW(ctx, nbPage, nbSeed, fmt.Sprintf("vwww/%d", nbPage))
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
			return vwww.NewVWWW(os.Args[3]).Serve(ctx)

		}

		return errors.New("invalid subcommand: generate or serve is expected")
	}

	return errors.New("invalid command: crawl or vwww is expected")
}
