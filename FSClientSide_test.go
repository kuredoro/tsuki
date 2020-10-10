package tsuki_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kureduro/tsuki"
)

func TestFSChunkSend(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage{
        Index: map[string]string {
            "0" : "Hello",
            "1" : "world",
        },
    }
    fsd := tsuki.NewFileServer(store)

    t.Run("get expected chunk 0",
    func (t *testing.T) {
        chunkId := "0"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionRead, chunkId, token)

        request := tsuki.NewGetChunkRequest(chunkId, token)
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index[chunkId])
    })

    t.Run("get expected chunk 1 twice",
    func (t *testing.T) {
        chunkId := "1"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionRead, chunkId, token)

        // Get chunk as expected
        request := tsuki.NewGetChunkRequest(chunkId, token)
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index[chunkId])

        // Another get wasn't expected
        request = tsuki.NewGetChunkRequest(chunkId, token)
        response = httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
    })

    t.Run("get expected chunk 1 with bad token and correct one",
    func (t *testing.T) {
        chunkId := "1"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionRead, chunkId, token)

        // Bad token first
        request := tsuki.NewGetChunkRequest("1", "xyz")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)

        // Correct token afterwards
        request = tsuki.NewGetChunkRequest("1", token)
        response = httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index[chunkId])
    })

    t.Run("get expected but unregistered chunk abc",
    func (t *testing.T) {
        chunkId := "abc"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionRead, chunkId, token)

        request := tsuki.NewGetChunkRequest("abc", token)
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotFound)
    })

    t.Run("get unexpected unregistered chunk",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("abc", "xyz")
        response := httptest.NewRecorder()

        fsd.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
    })
}

func TestFSChunkReceive(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage {
        Index : map[string]string {
            "0" : "abcde",
            "1" : "xyzw",
        },
    }

    server := tsuki.NewFileServer(store)

    token := "abcd"

    t.Run("upload chunk 2",
    func (t *testing.T) {
        text := "This is chunk 2"
        request := tsuki.NewPostChunkRequest("2", text, token)
        response := httptest.NewRecorder()

        server.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, "2", text)
    })

    t.Run("upload chunk 10",
    func (t *testing.T) {
        text := "another chunk"
        request := tsuki.NewPostChunkRequest("10", text, token)
        response := httptest.NewRecorder()

        server.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, "10", text)
    })

    t.Run("upload chunk 10 again",
    func (t *testing.T) {
        text := "changed chunk"
        request := tsuki.NewPostChunkRequest("10", text, token)
        response := httptest.NewRecorder()

        server.ServerClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
    })
}
