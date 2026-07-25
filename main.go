package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"filez/internal/server"
)

func main() {
	root := flag.String("root", "./data", "root directory to serve")
	addr := flag.String("addr", ":8080", "listen address")
	token := flag.String("token", "", "shared auth token (or set FILEZ_TOKEN)")
	flag.Parse()

	if *token == "" {
		*token = os.Getenv("FILEZ_TOKEN")
	}
	if *token == "" {
		log.Fatal("token required: pass --token or set FILEZ_TOKEN")
	}

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("resolve root: %v", err)
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		log.Fatalf("create root: %v", err)
	}

	srv := server.New(absRoot, *token)
	log.Printf("filez serving %s on %s", absRoot, *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		log.Fatal(err)
	}
}
