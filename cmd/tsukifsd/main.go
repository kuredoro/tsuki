package main

import (
	"flag"
	"fmt"
    "os"
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
    flag.StringVar(&ns, "ns", "", "address of the name server")
}

func main() {

    flag.Parse()

    if _, err := os.Stat(".tsukifs"); err == nil {
        save, err := os.Open(".tsukifs")
        if err != nil {
            log.Fatal(err)
        }
        defer save.Close()

        fmt.Fscanf(save, "%s", &ns)
        log.Printf("Found .tsukifs: NS=%s", ns)
    }

    addrForClients := ":" + strconv.Itoa(port)
    addrForInner := ":" + strconv.Itoa(port + 1)

    log.Printf("listening for clients at %s", addrForClients)

    store := &tsuki.InMemoryChunkStorage{
        Index : map[string]string {
            "0" : "hi",
            "1" : "how",
            "2" : "are",
            "3" : "you",
        },
    }

    nsConn := &tsuki.HTTPNSConnector{}
    nsConn.SetNSAddr(ns)

    heart := tsuki.NewHeart(nsConn, 3 * time.Second)
    go heart.Poll(-1)

    server := tsuki.NewFileServer(store, nsConn)

    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        if err := http.ListenAndServe(addrForClients, http.HandlerFunc(server.ServeClient)); err != nil {
            log.Fatalf("could not listen on %v, %v", addrForClients, err)
        }
    }()

    go func() {
        defer wg.Done()
        if err := http.ListenAndServe(addrForInner, http.HandlerFunc(server.ServeInner)); err != nil {
            log.Fatalf("could not listen on %v, %v", addrForClients, err)
        }
    }()

    wg.Wait()
}
