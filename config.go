package main

import (
	"github.com/BurntSushi/toml"
	"log"
)

// Config structure. Is used by tha mailer to send emails to right audience
// with the right content.
type Config struct {
	FlowerApiUrl string
	Server       string
	Port         int
	Email        string
	Receivers    []string
}

// Parses a config file and references config settings.
func (c *Config) Read() {
	if _, err := toml.DecodeFile("config/config.toml", &c); err != nil {
		log.Fatal(err)
	}
}
