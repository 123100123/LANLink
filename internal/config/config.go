package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	defaultPort                = "8787"
	defaultTransferChunkSize   = 64 * 1024
	defaultTransferMaxInFlight = 16
	defaultDiscoveryPort       = 8788
)

type Config struct {
	Port string

	TransferChunkSize         int
	TransferMaxInFlightChunks int

	// DiscoveryPort is the UDP port used by the LAN discovery beacon/scan.
	DiscoveryPort int
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return Config{
		Port: os.Getenv("LANLINK_PORT"),

		TransferChunkSize: envInt(
			"TRANSFER_CHUNK_SIZE",
			defaultTransferChunkSize,
		),

		TransferMaxInFlightChunks: envInt(
			"TRANSFER_MAX_IN_FLIGHT_CHUNKS",
			defaultTransferMaxInFlight,
		),

		DiscoveryPort: envInt(
			"LANLINK_DISCOVERY_PORT",
			defaultDiscoveryPort,
		),
	}.withDefaults()
}

func (c Config) withDefaults() Config {
	if c.Port == "" {
		c.Port = defaultPort
	}

	if c.TransferChunkSize <= 0 {
		c.TransferChunkSize = defaultTransferChunkSize
	}

	if c.TransferMaxInFlightChunks <= 0 {
		c.TransferMaxInFlightChunks = defaultTransferMaxInFlight
	}

	if c.DiscoveryPort <= 0 {
		c.DiscoveryPort = defaultDiscoveryPort
	}

	return c
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("invalid %s=%q, using default %d", key, raw, fallback)
		return fallback
	}

	if value <= 0 {
		log.Printf("invalid %s=%q, using default %d", key, raw, fallback)
		return fallback
	}

	return value
}
