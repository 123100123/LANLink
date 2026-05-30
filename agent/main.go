package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	port := os.Getenv("LANLINK_PORT")
	if port == "" {
		port = "8787"
	}

	http.HandleFunc("/health", healthHandler)

	address := ":" + port

	log.Println("LANLink agent listening on", address)

	err = http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal(err)
	}
}