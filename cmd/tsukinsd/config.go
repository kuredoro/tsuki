package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"time"
)

type Config struct {
	Namenode Namenode
	Storage  []storage
}

type Namenode struct {
	Host               string
	PublicPort         int
	PrivatePort        int
	TreeUpdatePeriod   int64
	TreeLogName        string
	TreeGobName        string
	ChunkTableGobName  string
	SoftDeathTime      time.Duration
	HardDeathTime      time.Duration
	ChunkSize          int
	Replicas           int
	StoragePrivatePort int
}

type storage struct {
	Host string
}

func LoadConfig() (*Config, error) {
	var conf *Config

	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		// shit
		return nil, fmt.Errorf("Config file is not found, %v", err)
	}

	return conf, nil
}
