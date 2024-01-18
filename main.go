package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	server, err := NewServer("debug/config.toml")
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err = http.ListenAndServe(":8080", server); err != nil {
		log.Fatalf("Server quit listening unexpectedly: %v", err)
	}

	os.Exit(0)
}
