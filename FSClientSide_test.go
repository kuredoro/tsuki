package tsuki_test

import (
    "fmt"
	"net/http"
	"net/http/httptest"
	"testing"

    "github.com/kureduro/tsuki"
)

type InMemoryChunkStorage struct {
    index map[int64]string
}

func (s *InMemoryChunkStorage) GetChunk(id int64) (string, error) {
    chunk, exists := s.index[id]

    if !exists {
        return "", fmt.Errorf("no chunk associated with id")
    }

    return chunk, nil
}

func TestFSChunkDownload(t *testing.T) {
    store := &InMemoryChunkStorage{
        index: map[int64]string {
            0 : "Hello",
            1 : "world",
        },
    }
    fsd := tsuki.NewFSClientSide(store)

    t.Run("get chunk 0",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("0")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.index[0])
    })

    t.Run("get chunk 1",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("1")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.index[1])
    })

    t.Run("get undefined chunk 2",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("2")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotFound)
    })

    t.Run("get invalid chunk abc",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("abc")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotAcceptable)
    })

    t.Run("get invalid chunk -1",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("-1")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotAcceptable)
    })

    t.Run("get invalid chunk 123456789123456789123456789",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("123456789123456789123456789")
        response := httptest.NewRecorder()

        fsd.ServerHTTP(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotAcceptable)
    })
}
