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

	"github.com/TheBigRoomXXL/backlinks-engine/internal/vwww"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("A command (crawl or vwww) is expected as argument")
	}

	cmd := os.Args[1]
	if cmd != "crawl" && cmd != "vwww" {
		log.Fatal("Invalid command: crawl or vwww is expected")
	}
	if cmd == "crawl" {
		// internal.Crawl(s, db, os.Args[2:])
		log.Fatal("Not Implemented")
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
			vwww.GenerateVWWW(nbPage, nbSeed, fmt.Sprintf("vwww/%d", nbPage))
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
			err := vwww.NewVWWW(os.Args[3]).Serve()
			if err != nil {
				log.Fatal("VWWW crashed:", err)
			}

		}
	}
}
