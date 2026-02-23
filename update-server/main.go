package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to the YAML config file")
	flag.Parse()

	store, err := NewConfigStore(*configPath)
	if err != nil {
		log.Fatalf("failed to load config %q: %v", *configPath, err)
	}

	if err := store.Watch(); err != nil {
		// Non-fatal: the server still works, rules just won't hot-reload.
		log.Printf("warning: could not watch config file for changes: %v", err)
	}

	cfg := store.Get()
	port := cfg.Server.Port
	if port == 0 {
		port = 8080
	}

	mux := http.NewServeMux()
	mux.Handle("/update/", NewHandler(store))

	addr := fmt.Sprintf(":%d", port)
	log.Printf("update server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
