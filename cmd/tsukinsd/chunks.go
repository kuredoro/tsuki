package main

import (
	"encoding/gob"
	"fmt"
	"os"
)

const (
	PENDING  = 0
	OK       = 1
	OBSOLETE = 2
)

type Chunk struct {
	ChunkID string
	File    string
	//Nodes         []*FileServerInfo
	Nodes         map[string]*FileServerInfo
	Status        []int
	Statuses      map[string]int
	ReadyReplicas int
	AllReplicas   int
}

type ChunkTable struct {
	Table         map[string]*Chunk
	InvertedTable map[string][]*Chunk // node hostname -> []*Chunk
}

func (ct *ChunkTable) AddChunk(chunkID string, file string, initNode *FileServerInfo) bool {
	chunk := Chunk{
		ChunkID:     chunkID,
		File:        file,
		Nodes:       map[string]*FileServerInfo{initNode.Host: initNode},
		Status:      append([]int{PENDING}),
		Statuses:    map[string]int{initNode.Host: PENDING},
		AllReplicas: 1,
	}

	ct.Table[chunkID] = &chunk

	return true
}

func (c *Chunk) AddFSToChunk(fs *FileServerInfo) {
	c.Nodes[fs.Host] = fs
	c.Statuses[fs.Host] = PENDING
}

func (ct *ChunkTable) SaveChunkTable(saveTo string) bool {
	file, _ := os.Create(saveTo)
	defer file.Close()
	encoder := gob.NewEncoder(file)

	encoder.Encode(ct)
	return true
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
	return fmt.Sprintf("Chunk{ChunkID: %s, File: %s, Nodes: %v, Status: %d}", c.ChunkID, c.File, c.Nodes, c.Status)
}
