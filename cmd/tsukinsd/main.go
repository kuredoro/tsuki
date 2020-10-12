package main

import (
	"encoding/gob"
	"log"
	"os"
)

type ChunkMessage struct {
	ChunkID   string `json:"chunkID"`
	StorageIP string `json:"storageIP"`
}

type ClientMessage struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Objects []string       `json:"objects"`
	Token   string         `json:"token"`
	Chunks  []ChunkMessage `json:"chunks"`
}

var t *Tree
var conf *Config
var storages *PoolInfo
var ct = &ChunkTable{
	Table:         map[string]*Chunk{}, // chunkID -> chunk
	InvertedTable: map[string][]*Chunk{},
}

var tokens = map[string][]*FileServerInfo{}

func loadAll() {
	file, _ := os.Open("t.gob")
	defer file.Close()
	decoder := gob.NewDecoder(file)
	decoder.Decode(t)

	file, _ = os.Open("st.gob")
	defer file.Close()
	decoder = gob.NewDecoder(file)
	decoder.Decode(storages)

	file, _ = os.Open("ct.gob")
	defer file.Close()
	decoder = gob.NewDecoder(file)
	decoder.Decode(ct)
}

func saveAll() {
	file, _ := os.Create("t.gob")
	defer file.Close()
	decoder := gob.NewEncoder(file)
	decoder.Encode(t)

	file, _ = os.Create("st.gob")
	defer file.Close()
	decoder = gob.NewEncoder(file)
	decoder.Encode(storages)

	file, _ = os.Create("ct.gob")
	defer file.Close()
	decoder = gob.NewEncoder(file)
	decoder.Encode(ct)
}

func main() {
	var err error
	conf, err = LoadConfig()

	if err != nil {
		log.Fatal(err)
	}

	//t = LoadTree(conf.Namenode.TreeGobName)
	t = InitTree(conf.Namenode)
	storages = InitFServers(conf)
	//loadAll()

	go StartPrivateServer()
	go storages.HeartbeatManager(true)
	go storages.HeartbeatManager(false)
	StartPublicServer()
}
