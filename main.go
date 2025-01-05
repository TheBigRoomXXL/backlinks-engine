package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal/client"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/commons"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/crawler"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/exporter"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/queue"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/robot"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/settings"
	"github.com/TheBigRoomXXL/backlinks-engine/internal/storage"
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
	// Hacky debug flag
	if len(os.Args) > 2 && os.Args[1] == "--debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}

	if len(os.Args) < 2 {
		return errors.New("a command (crawl or vwww) is expected as argument")
	}

	cmd := os.Args[1]

	if cmd == "crawl" {
		s, ok := settings.New()
		if !ok {
			return errors.New("failed to initialize setttings properly")
		}

		PostgresURI := fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?%s",
			s.DB_USER,
			s.DB_PASSWORD,
			s.DB_HOSTNAME,
			s.DB_PORT,
			s.DB_NAME,
			s.DB_OPTIONS,
		)

		postgres, err := storage.NewPostgres(ctx, PostgresURI)
		if err != nil {
			return fmt.Errorf("failed init postgres connection pool: %w", err)
		}
		queue := queue.NewFIFOQueue()
		exporter := exporter.NewPostgresExporter(postgres)
		fetcher := client.NewCrawlClient(ctx, s.HTTP_RATE_LIMIT, s.HTTP_MAX_RETRY, s.HTTP_TIMEOUT)
		robot := robot.NewInMemoryRobotPolicy(fetcher)
		crawler := crawler.NewCrawler(ctx, queue, fetcher, robot, exporter, s.CRAWLER_MAX_CONCURENCY)

		seeds, err := parseSeeds(os.Args[2:])
		if err != nil {
			return fmt.Errorf("fail to parse argument: %w", err)
		}
		for _, seed := range seeds {
			crawler.AddUrl(seed)
		}

		go telemetry.MetricsReport(ctx)
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

func parseSeeds(args []string) ([]*url.URL, error) {
	seeds := make([]*url.URL, 0)
	for _, arg := range args {
		_, error := os.Stat(arg)
		if errors.Is(error, os.ErrNotExist) {
			url, err := url.Parse(arg)
			if err != nil {
				return nil, fmt.Errorf("failed to parsed seed: %w", err)
			}
			url, err = commons.NormalizeUrl(url)
			if err != nil {
				return nil, fmt.Errorf("failed to normalize seed: %w", err)
			}
			seeds = append(seeds, url)
		} else {
			file, err := os.Open(arg)
			if err != nil {
				return nil, fmt.Errorf("error opening input file: %s", err)
			}
			input, err := io.ReadAll(file)
			if err != nil {
				return nil, fmt.Errorf("error reading input file: %s", err)
			}
			for _, seed := range strings.Fields(string(input)) {
				url, err := url.Parse(seed)
				if err != nil {
					return nil, fmt.Errorf("failed to parsed seed: %w", err)
				}
				url, err = commons.NormalizeUrl(url)
				if err != nil {
					return nil, fmt.Errorf("failed to normalize seed: %w", err)
				}
				seeds = append(seeds, url)
			}
		}

	}
	return seeds, nil
}
