package models

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config implements configuration functionality
type Config struct {
	Databases map[string]DBConf `toml:"databases"`
	Secrets   map[string]string `toml:"secrets"`
}

// Load loads the config from the config file
func (c *Config) Load() (ok bool) {
	ok = true
	configfile := "config.toml"
	_, err := os.Stat(configfile)
	if err != nil {
		ok = false
		// ErrorLogger.Fatal("Config file not found, program exiting.\n", err)
	}

	if _, err := toml.DecodeFile(configfile, c); err != nil {

		fmt.Println(err.Error())
		ok = false
		// ErrorLogger.Fatal("Config file could not be decoded, program exiting.\n")
	}

	return ok
}
