package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/TheBigRoomXXL/backlinks-engine/internal"
)

func main() {
	s := internal.NewSettings()

	db, err := internal.NewDatabase(s)
	if err != nil {
		log.Fatal(err)
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
		internal.Crawl(s, db, os.Args[2:])
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
			vwww := internal.NewVWWW(nbPage, nbSeed)
			t1 := time.Now()
			err = vwww.DumpCSV(fmt.Sprintf("vwww/%d", nbPage))
			t2 := time.Now()
			fmt.Println("Time to generate:", t1.Sub(t0))
			fmt.Println("Time to dump:", t2.Sub(t1))
			if err != nil {
				log.Fatal("failed to dump VWWW to CVS:", err)
			}
			return
		}
		if subcmd == "serve" {
			if len(os.Args) < 4 {
				log.Fatal("serve expect a path to a dumped vwww")
			}
			vwww, err := internal.NewVWWWFromCSV(os.Args[3])
			if err != nil {
				log.Fatal("failed to load VWWW:", err)
			}
			err = vwww.Serve()
			if err != nil {
				log.Fatal("failed to load VWWW:", err)
			}
		}
	}
}
