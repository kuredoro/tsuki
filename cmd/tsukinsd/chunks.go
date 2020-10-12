package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"sync"
)

const (
	PENDING  = 0
	OK       = 1
	OBSOLETE = 2
	DOWN     = 3
)

type Chunk struct {
	ChunkID string
	File    string
	//FServers         []*FileServerInfo
	FServers      map[string]*FileServerInfo
	Status        int
	Statuses      map[string]int
	ReadyReplicas int
	AllReplicas   int
	ssmu          sync.Mutex
}

type ChunkTable struct {
	ivmu          sync.Mutex
	Table         map[string]*Chunk
	InvertedTable map[string][]*Chunk // node hostname -> []*Chunk
}

func (ct *ChunkTable) AddChunk(chunkID string, file string, initNode *FileServerInfo) (*Chunk, bool) {
	chunk := Chunk{
		ChunkID:     chunkID,
		File:        file,
		FServers:    map[string]*FileServerInfo{initNode.PrivateHost: initNode},
		Status:      PENDING,
		Statuses:    map[string]int{initNode.PrivateHost: PENDING},
		AllReplicas: 1,
	}

	ct.Table[chunkID] = &chunk

	return &chunk, true
}

func (c *Chunk) AddFSToChunk(fs *FileServerInfo) {
	c.FServers[fs.PrivateHost] = fs
	c.Statuses[fs.PrivateHost] = PENDING
	c.AllReplicas += 1
}

func (ct *ChunkTable) SaveChunkTable(saveTo string) bool {
	file, _ := os.Create(saveTo)
	defer file.Close()
	encoder := gob.NewEncoder(file)

	encoder.Encode(ct)
	return true
}

func (ct *ChunkTable) PurgeChunks(chunks []string) {
	cock := map[int][]string{}
	for _, chunkName := range chunks {
		chunk := ct.Table[chunkName]

		chunk.Status = OBSOLETE
		for _, fs := range chunk.FServers {
			if fs.GetStatus() == LIVE {
				// todo: add to queue if not alive
				cock[fs.ID] = append(cock[fs.ID], chunk.ChunkID)
			}
		}
	}

	for key, value := range cock {
		storages.PurgeChunks(key, value)
	}
}

func LoadChunkTable(openFrom string) *ChunkTable {
	file, _ := os.Open(openFrom)
	defer file.Close()

	decoder := gob.NewDecoder(file)

	var chunkTable ChunkTable
	decoder.Decode(&chunkTable)

	return &chunkTable
}

func (ct *ChunkTable) String() string {
	return fmt.Sprintf("ChunkTable{ChunkTable: %v}", ct.Table)
}

func (c *Chunk) String() string {
	return fmt.Sprintf("Chunk{ChunkID: %s, File: %s, FServers: %v, Status: %d}", c.ChunkID, c.File, c.FServers, c.Status)
}

func (c *Chunk) SetStatus(status int) {
	c.ssmu.Lock()
	defer c.ssmu.Unlock()
	c.Status = status
}
