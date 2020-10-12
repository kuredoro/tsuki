package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func initTree(w http.ResponseWriter, r *http.Request) {
	t = InitTree(conf.Namenode)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ClientMessage{Status: "OK", Message: "The tree is initialized"})
}



//func confirmChunk(w http.ResponseWriter, r *http.Request) {
//	file :=
//}

var t *Tree
var conf *Config
var storages *PoolInfo
var ct = &ChunkTable{
	Table: map[string]*Chunk{}, // chunkID -> chunk
	InvertedTable: map[string][]*Chunk{},
}

var tokens = map[string][]*FileServerInfo{}

func main() {
	var err error
	conf, err = LoadConfig()

	if err != nil {
		log.Fatal(err)
	}

	var sos = map[string][]string{}

	fmt.Printf("\n%v\n", sos["cock"])

	//t = LoadTree(conf.Namenode.TreeGobName)
	t = InitTree(conf.Namenode)
	storages = InitFServers(conf)
	//go startPulse()
	go StartPrivateServer()
	go storages.HeartbeatManager(true)
	go storages.HeartbeatManager(false)
	StartPublicServer()



	//r.HandleFunc("/confirmChunk", confirmChunk).Methods("GET") // not forever
}
