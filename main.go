package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal"
)

func main() {
	s := internal.NewSettings()

	db, err := internal.NewDatabase(s)
	if err != nil {
		log.Fatal("failed to init db: ", err)
	}
	defer db.Close()

	if len(os.Args) < 2 {
		log.Fatal("A command (crawl or vwww) is expected as argument")
	}

	cmd := os.Args[1]
	if cmd != "crawl" && cmd != "vwww" {
		log.Fatal("Invalid command: crawl or vwww is expected")
	}
	if cmd == "crawl" {
		seeds := make([]string, 0)
		for _, arg := range os.Args[2:] {
			_, error := os.Stat(arg)
			if errors.Is(error, os.ErrNotExist) {
				seed, err := internal.NormalizeUrlString(arg)
				if err != nil {
					log.Fatalf("failed to normalize seed: %s", err)
				}
				seeds = append(seeds, seed)
			} else {
				file, err := os.Open(arg)
				if err != nil {
					log.Fatalf("error opening input file: %s", err)
				}
				input, err := io.ReadAll(file)
				if err != nil {
					log.Fatalf("error reading input file: %s", err)
				}
				for _, arg := range strings.Fields(string(input)) {
					seed, err := internal.NormalizeUrlString(arg)
					if err != nil {
						log.Fatalf("failed to normalize seed: %s", err)
					}
					seeds = append(seeds, seed)
				}
			}

		}
		internal.Crawl(s, db, seeds)
	}
	if cmd == "vwww" {
		if len(os.Args) < 3 {
			log.Fatal("vwww expect a subcommand (generate or serve) as argument")
		}

		subcmd := os.Args[2]
		if subcmd != "generate" && subcmd != "serve" {
			log.Fatal("Invalid subcommand: generate or serve is expected")
		}

		if subcmd == "generate" {
			if len(os.Args) < 5 {
				log.Fatal("generate expect 2 argument: nbPage and nbSeed")
			}
			nbPage, err := strconv.Atoi(os.Args[3])
			if err != nil {
				log.Fatal("failed to parse nbPage:", err)
			}
			nbSeed, err := strconv.Atoi(os.Args[4])
			if err != nil {
				log.Fatal("failed to parse nbSeed:", err)
			}
			t0 := time.Now()
			internal.GenerateVWWW(nbPage, nbSeed, fmt.Sprintf("vwww/%d", nbPage))
			if err != nil {
				log.Fatal("failed to dump VWWW to CVS:", err)
			}
			fmt.Println("Time to generate:", time.Since(t0))
			return
		}
		if subcmd == "serve" {
			if len(os.Args) < 4 {
				log.Fatal("serve expect a path to a dumped vwww")
			}
			err := internal.NewVWWW(os.Args[3]).Serve()
			if err != nil {
				log.Fatal("VWWW crashed:", err)
			}

		}
	}
}
