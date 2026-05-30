package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	port := os.Getenv("LANLINK_PORT")
	if port == "" {
		port = "8787"
	}

	return Config{
		Port: port,
	}
}