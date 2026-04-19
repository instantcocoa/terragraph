package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/terragraph/backend/internal/api"
)

func main() {
	port := flag.Int("port", 3001, "server port")
	flag.Parse()

	server := api.NewServer()
	addr := fmt.Sprintf(":%d", *port)

	log.Printf("TerraGraph API server starting on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
