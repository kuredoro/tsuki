package tsuki_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kureduro/tsuki"
)

func TestFS_ChunkSend(t *testing.T) {
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

        fsd.ServeClient(response, request)

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

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertResponseBody(t, response.Body.String(), store.Index[chunkId])

        // Another get wasn't expected
        request = tsuki.NewGetChunkRequest(chunkId, token)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
    })

    t.Run("get expected chunk 1 with bad token and correct one",
    func (t *testing.T) {
        chunkId := "1"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionRead, chunkId, token)

        // Bad token first
        request := tsuki.NewGetChunkRequest("1", "xyz")
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)

        // Correct token afterwards
        request = tsuki.NewGetChunkRequest("1", token)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

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

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusNotFound)
    })

    t.Run("get unexpected unregistered chunk",
    func (t *testing.T) {
        request := tsuki.NewGetChunkRequest("abc", "xyz")
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
    })
}

func TestFS_ChunkReceive(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage {
        Index : map[string]string {
            "0" : "abcde",
            "1" : "xyzw",
        },
    }

    fsd := tsuki.NewFileServer(store)

    t.Run("upload expected chunk 2",
    func (t *testing.T) {
        chunkId := "2"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionWrite, chunkId, token)

        text := "This is chunk 2"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, text)
    })

    t.Run("upload expected chunk 3 twice",
    func (t *testing.T) {
        chunkId := "3"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionWrite, chunkId, token)

        text := "test test foo bar"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, text)

        // Can not write twice
        request = tsuki.NewPostChunkRequest(chunkId, text, token)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
    })

    t.Run("upload unexpected chunk 4",
    func (t *testing.T) {
        chunkId := "4"
        token := chunkId

        text := "didn't expect me!?"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
    })

    t.Run("upload expected, but registered chunk 1",
    func (t *testing.T) {
        chunkId := "1"
        token := chunkId
        fsd.Expect(tsuki.ExpectActionWrite, chunkId, token)

        text := "i'm overwritting existing chunk!"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
    })
}

func TestFS_ReceiveExpect(t *testing.T) {
    store := &tsuki.InMemoryChunkStorage {
        Index: map[string]string {
            "0": "abracadabra",
            "1": "watashihanekodesuka",
        },
    }

    fsd := tsuki.NewFileServer(store)

    token := "abc"

    batch1 := []string{ "a", "b", "c" }
    want := tsuki.ExpectActionRead

    request := tsuki.NewExpectRequest("read", token, batch1)
    response := httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusOK)
    for _, id := range batch1 {
        got := fsd.GetTokenExpectationForChunk(token, id)
        if  got != want {
            t.Errorf("1st request: token=%s chunk=%s, got action %v, want %v", token, id, got, want)
        }
    }

    batch2 := []string{ "b", "c", "d" }
    want = tsuki.ExpectActionWrite

    request = tsuki.NewExpectRequest("write", token, batch2)
    response = httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusOK)
    for _, id := range batch2 {
        got := fsd.GetTokenExpectationForChunk(token, id)
        if  got != want {
            t.Errorf("2st request: token=%s chunk=%s, got action %v, want %v", token, id, got, want)
        }
    }
}
