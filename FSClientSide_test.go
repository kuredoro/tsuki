package tsuki_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kureduro/tsuki"
)

func TestFSChunkDownload(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage{
        Index: map[string]string {
            "0" : "Hello",
            "1" : "world",
        },
    }
    fsd := tsuki.NewFileServer(store)

    t.Run("get chunk 0",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("0")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index["0"])
    })

    t.Run("get chunk 1",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("1")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index["1"])
    })

    t.Run("get undefined chunk 2",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("2")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotFound)
    })

    t.Run("get undefined chunk abc",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("abc")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotFound)
    })
}

func TestFSChunkUpload(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage {
        Index : map[string]string {
            "0" : "abcde",
            "1" : "xyzw",
        },
    }

    server := tsuki.NewFileServer(store)

    t.Run("upload chunk 2",
    func (t *testing.T) {
        text := "This is chunk 2"
        request := tsuki.NewPostChunkRequest("2", text)
        response := httptest.NewRecorder()

        server.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, "2", text)
    })

    /*
    t.Run("upload chunk 10",
    func (t *testing.T) {
        text := "another chunk"
        request := tsuki.NewPostChunkRequest("10", text)
        response := httptest.NewRecorder()

        server.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, "10", text)
    })
    */
}
