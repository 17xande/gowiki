package main

import (
	"encoding/json"
	"os"
)

// Config represents the applications configuration
type Config struct {
	Database DB
}

var config Config

func loadConfig() {
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	_ = decoder.Decode(&config)
}
