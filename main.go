package main

import (
	"log"
	"os"

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
		vwww := internal.NewVirtualWorldWideWeb(1_000)
		internal.ServeVWWW(vwww)
	}
}
