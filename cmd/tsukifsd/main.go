package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/kureduro/tsuki"
)

var port int
var ns string

func init() {
    flag.IntVar(&port, "port", 7000, "port for clients")
    flag.StringVar(&ns, "ns", "http://localhost:7001", "address of the name server")
}

func main() {

    flag.Parse()

    addrForClients := ":" + strconv.Itoa(port)
    addrForInner := ":" + strconv.Itoa(port + 1)

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

    /*
    nsConn := &tsuki.HTTPNSConnector{ addr: ns }
    */

    server := tsuki.NewFileServer(store, nil)

    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        if err := http.ListenAndServe(addrForClients, http.HandlerFunc(server.ServeClient)); err != nil {
            log.Fatalf("could not listen on %v, %v", addrForClients, err)
        }
    }()

    go func() , nil{
        defer wg.Done()
        if err := http.ListenAndServe(addrForInner, http.HandlerFunc(server.ServeInner)); err != nil {
            log.Fatalf("could not listen on %v, %v", addrForClients, err)
        }
    }()

    wg.Wait()
}
