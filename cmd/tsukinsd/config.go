package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Namenode namenode
	Storage  []storage
}

type namenode struct {
	Host string
	Port int
}

type storage struct {
	Host string
	Port int 
}

func LoadConfig() (*Config, error) {
	var conf *Config

	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		// shit
		return nil, fmt.Errorf("Config file is not found, %v", err)
	}

	return conf, nil
}
