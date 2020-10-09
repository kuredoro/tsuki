package tsuki_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

    "github.com/kureduro/tsuki"
)

type InMemoryChunkStorage struct {
    index map[int]string
}

func TestFSChunkDownload(t *testing.T) {
    chunk0 := "Hello!"

    store := &InMemoryChunkStorage{
        index: map[int]string {
            0 : chunk0,
        },
    }
    fsd := tsuki.NewFSClientSide(store)

    request, _ := http.NewRequest(http.MethodGet, "/chunks/0", nil)
    response := httptest.NewRecorder()

    fsd.ServerHTTP(response, request)

    if response.Body.String() != chunk0 {
        t.Errorf("got chunk contents %q, want %q", response.Body.String(), chunk0)
    }
}
