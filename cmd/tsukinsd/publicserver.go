package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"math"
	"net/http"
	"strconv"
)

func ls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	list, err := t.LS(address)
	if err == nil {
		json.NewEncoder(w).Encode(&ClientMessage{
			Status:  "OK",
			Message: fmt.Sprintf("The content of %s", address),
			Objects: list})
	} else {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
	}
}

func mkdir(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dirName := r.URL.Query().Get("address")
	err := t.CreateDirectory(dirName)
	if err == nil {
		json.NewEncoder(w).Encode(&ClientMessage{
			Status:  "OK",
			Message: fmt.Sprintf("%d %s directory successfullly created", t.Version, dirName)})
	} else {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
	}
}

func touch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	_, err := t.CreateFile(address)
	if err == nil {
		json.NewEncoder(w).Encode(&ClientMessage{
			Status:  "OK",
			Message: fmt.Sprintf("%d %s file successfullly created", t.Version, address)})
	} else {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
	}
}

func cd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	address, err := t.CD(address)
	if err == nil {
		json.NewEncoder(w).Encode(&ClientMessage{
			Status:  "OK",
			Message: fmt.Sprintf("You can change directory to %s", address),
			Objects: []string{address},
		})
	} else {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
	}
}

func upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sizeStr := r.URL.Query().Get("size")
	address := r.URL.Query().Get("address")

	size, err := strconv.ParseInt(sizeStr, 10, 64)

	if err != nil {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
		return
	}

	file, err := t.CreateFile(address)
	if err != nil {
		json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
		return
	}

	chunkNum := int(math.Ceil(float64(size) / 1024 / 1024 / float64(conf.Namenode.ChunkSize)))
	var chunks []ChunkMessage

	token := generateToken()

	//fmt.Printf("%q", string(tokenBytes))

	inversed := map[string][]string{}

	for i := 0; i < chunkNum; i++ {
		chunkID, _ := uuid.NewUUID()
		//fmt.Printf("%s\n", chunkID.String())

		storageNode := storages.Select()
		chunks = append(chunks, ChunkMessage{ChunkID: chunkID.String(), StorageIP: storageNode.Host})

		file.Chunks = append(file.Chunks, chunkID.String())
		file.Pending[chunkID.String()] = true

		ct.AddChunk(chunkID.String(), file.Address, storageNode)
		fmt.Printf("%v\n", ct)
		address := fmt.Sprintf("%s:%d", storageNode.Host, storageNode.Port)

		inversed[address] = append(inversed[address], chunkID.String())
	}

	//fmt.Printf("%v", inversed)
	//fmt.Printf("%v\n", t)
	//fmt.Printf("%v\n", ct)
	json.NewEncoder(w).Encode(&ClientMessage{Status: "OK", Message: "Go upload there", Chunks: chunks, Token: token})

	go SendChunksToFS(inversed, token)
	// requests to fs's /expect/write?token JSON {chunks: []int}
	// confirmation from fs's /confirm?chunkID=<chunkID>
	// or client says /fserror?token=<token> <- for now error on client
	// small ttl for client ~12 secs, big for ns - 180 secs
	// if there is any error -> cancel the previous operation and restart it

	// everything is ok
	// fs works like client now
}

func reupload(w http.ResponseWriter, r *http.Request) {
	// todo
}



func StartPublicServer() {
	r := mux.NewRouter()
	r.HandleFunc("/init", initTree).Methods("GET")
	r.HandleFunc("/ls", ls).Methods("GET")
	r.HandleFunc("/mkdir", mkdir).Methods("GET")
	r.HandleFunc("/touch", touch).Methods("GET")
	r.HandleFunc("/cd", cd).Methods("GET")
	r.HandleFunc("/upload", upload).Methods("GET")
	r.HandleFunc("/reupload", reupload).Methods("GET")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", conf.Namenode.PublicPort), r))
}