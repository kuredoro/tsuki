package tsuki

import (
    "fmt"
    "net/http"
    "log"
)

type NSConnector interface {
    ReceivedChunk(id string)

    SetNSAddr(addr string)
    GetNSAddr() string
    IsNS(addr string) bool

    Poller
}

type HTTPNSConnector struct {
    Addr string
}

func (c *HTTPNSConnector) ReceivedChunk(id string) {
    url := fmt.Sprintf("%s/confirm/receivedChunk?chunkID=%s", c.Addr, id)
    log.Printf("ReceivedChunk: %s", url)
    go http.Get(url)
}

func (c *HTTPNSConnector) GetNSAddr() string {
    return c.Addr
}

func (c *HTTPNSConnector) SetNSAddr(addr string) {
    c.Addr = addr
}

func (c *HTTPNSConnector) IsNS(addr string) bool {
    if c.Addr == "" {
        return true
    }

    return c.Addr == addr
}

func (c *HTTPNSConnector) Poll() {
    go http.Get(c.Addr + "/pulse")
}


type SpyNSConnector struct {
    receivedChunks []string
    Addr string
    PulseCount int
}

func (c *SpyNSConnector) ReceivedChunk(id string) {
    c.receivedChunks = append(c.receivedChunks, id)
}

func (c *SpyNSConnector) Reset() {
    c.receivedChunks = nil
}

func (c *SpyNSConnector) GetNSAddr() string {
    return c.Addr
}

func (c *SpyNSConnector) SetNSAddr(addr string) {
    c.Addr = addr
}

func (c *SpyNSConnector) IsNS(addr string) bool {
    if c.Addr == "" {
        return true
    }

    return c.Addr == addr
}

func (c *SpyNSConnector) Poll() {
    if c.Addr != "" {
        c.PulseCount++
    }       
}
