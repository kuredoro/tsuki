package tsuki

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ChunkStore interface {
    GetChunk(id int64) (string, error)
}

type FSClientSide struct {
    store ChunkStore
}

func NewFSClientSide(store ChunkStore) *FSClientSide {
    return &FSClientSide{
        store: store,
    }
}

func (cs *FSClientSide) ServerHTTP(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/chunks/")
    
    chunkId, err := strconv.Atoi(idStr)

    if err != nil || chunkId < 0{
        w.WriteHeader(http.StatusNotAcceptable)
        return
    }

    chunk, err := cs.store.GetChunk(int64(chunkId))

    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    fmt.Fprint(w, chunk)
    return
}
