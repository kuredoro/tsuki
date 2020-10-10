package tsuki

import (
    "fmt"
    "net/http"
    "log"
)

type HTTPNSConnector struct {
    Addr string
}

func (c *HTTPNSConnector) ReceivedChunk(id string) {
    url := fmt.Sprintf("%s/confirm/receivedChunk?chunkID=%s", c.Addr, id)
    log.Printf("ReceivedChunk: %s", url)
    go http.Get(url)
}

type SpyNSConnector struct {
    receivedChunks []string
}

func (c *SpyNSConnector) ReceivedChunk(id string) {
    c.receivedChunks = append(c.receivedChunks, id)
}

func (c *SpyNSConnector) Reset() {
    c.receivedChunks = nil
}
