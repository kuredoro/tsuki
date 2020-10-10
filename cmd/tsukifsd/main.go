package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kureduro/tsuki"
)

var addr = "localhost:7070"

func main() {

    if len(os.Args) == 1 {
        log.Printf("Using default address %v, you can pass the desired address as the second argument.", addr)
    }

    if len(os.Args) == 2 {
        addr = os.Args[1]
    }

    if len(os.Args) > 2 {
        log.Printf("warning: Got additional (count=%d) unneded command line arguments.", len(os.Args))
    }

    store := &tsuki.InMemoryChunkStorage{
        Index : map[int64]string {
            0 : "hi",
            1 : "how",
            2 : "are",
            3 : "you",
        },
    }

    server := tsuki.NewFileServer(store)
    if err := http.ListenAndServe(addr, http.HandlerFunc(server.ServerClient)); err != nil {
        log.Fatalf("could not listen on %v, %v", addr, err)
    }
}
