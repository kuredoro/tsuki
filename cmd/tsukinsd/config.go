package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Namenode Namenode
	Storage  []storage
}

type Namenode struct {
	Host string
	Port int
	TreeUpdatePeriod int64
	TreeLogName string
	TreeGobName string
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
