package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/kureduro/tsuki"
)

var clientPort int
var ns string

func init() {
    flag.IntVar(&clientPort, "client-port", 7000, "port for clients")
    flag.StringVar(&ns, "ns", "localhost:7001", "address of the name server")
}

func main() {

    flag.Parse()

    addrForClients := "localhost:" + strconv.Itoa(clientPort)

    log.Printf("listening for clients at %s", addrForClients)

    heart := tsuki.NewHeart(ns + "/pulse", 3 * time.Second)
    go heart.Poll(-1)

    store := &tsuki.InMemoryChunkStorage{
        Index : map[string]string {
            "0" : "hi",
            "1" : "how",
            "2" : "are",
            "3" : "you",
        },
    }

    server := tsuki.NewFileServer(store)
    if err := http.ListenAndServe(addrForClients, http.HandlerFunc(server.ServeClient)); err != nil {
        log.Fatalf("could not listen on %v, %v", addrForClients, err)
    }
}
