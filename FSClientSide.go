package tsuki

import (
	"fmt"
	"net/http"
)

type ChunkStore interface {

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
    fmt.Fprint(w, "Hello!")
    return
}
