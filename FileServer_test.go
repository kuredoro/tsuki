package tsuki_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log"
	"io/ioutil"
	"os"

	"github.com/kureduro/tsuki"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestFS_ChunkSend(t *testing.T) {
    store := tsuki.NewInMemoryChunkStorage(
        map[string]string {
            "0" : "Hello",
            "1" : "world",
    })

    nsConn := &tsuki.SpyNSConnector{}

    fsd := tsuki.NewFileServer(store, nsConn)

    t.Run("get expected chunk 0",
    func (t *testing.T) {
        chunkId := "0"
        token := chunkId
        fsd.Expect(token, tsuki.ExpectActionRead, chunkId)

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
        fsd.Expect(token, tsuki.ExpectActionRead, chunkId)

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
        fsd.Expect(token, tsuki.ExpectActionRead, chunkId)

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
        fsd.Expect(token, tsuki.ExpectActionRead, chunkId)

        request := tsuki.NewGetChunkRequest("abc", token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        // The server should not allow such expects to pass...
        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
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
    store := tsuki.NewInMemoryChunkStorage(
        map[string]string {
            "0" : "abcde",
            "1" : "xyzw",
    })

    nsConn := &tsuki.SpyNSConnector{}

    fsd := tsuki.NewFileServer(store, nsConn)

    t.Run("upload expected chunk 2",
    func (t *testing.T) {
        chunkId := "2"
        token := chunkId
        fsd.Expect(token, tsuki.ExpectActionWrite, chunkId)

        text := "This is chunk 2"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, text)
        tsuki.AssertReceivedChunkCalls(t, nsConn, chunkId)
    })

    t.Run("upload expected chunk 3 twice",
    func (t *testing.T) {
        nsConn.Reset()
        chunkId := "3"
        token := chunkId
        fsd.Expect(token, tsuki.ExpectActionWrite, chunkId)

        text := "test test foo bar"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, text)
        tsuki.AssertReceivedChunkCalls(t, nsConn, chunkId)

        // Can not write twice
        request = tsuki.NewPostChunkRequest(chunkId, text, token)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
        tsuki.AssertReceivedChunkCalls(t, nsConn, chunkId)
    })

    t.Run("upload unexpected chunk 4",
    func (t *testing.T) {
        nsConn.Reset()
        chunkId := "4"
        token := chunkId

        text := "didn't expect me!?"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
        tsuki.AssertReceivedChunkCalls(t, nsConn)
    })

    t.Run("upload expected, but already present chunk 1",
    func (t *testing.T) {
        nsConn.Reset()
        chunkId := "1"
        token := chunkId
        fsd.Expect(token, tsuki.ExpectActionWrite, chunkId)

        text := "i'm overwritting existing chunk!"
        request := tsuki.NewPostChunkRequest(chunkId, text, token)
        response := httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
        tsuki.AssertReceivedChunkCalls(t, nsConn)
    })
}

func TestFS_ReceiveExpect(t *testing.T) {
    store := tsuki.NewInMemoryChunkStorage(
        map[string]string {
            "a": "abracadabra",
            "b": "watashihanekodesuka",
            "c": "kimimonekodesuka",
    })

    nsConn := &tsuki.SpyNSConnector{}

    fsd := tsuki.NewFileServer(store, nsConn)

    token := "abc"

    batch1 := []string{ "a", "b", "c" }
    want := tsuki.ExpectActionRead

    request := tsuki.NewExpectRequest("read", token, batch1...)
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

    request = tsuki.NewExpectRequest("write", token, batch2...)
    response = httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusForbidden)
    for _, id := range batch1 {
        got := fsd.GetTokenExpectationForChunk(token, id)
        if  got != want {
            t.Errorf("2st request: token=%s chunk=%s, got action %v, want %v", token, id, got, want)
        }
    }
}

func TestFS_CancelExpect(t *testing.T) {
    store := tsuki.NewInMemoryChunkStorage(
        make(map[string]string),
    )

    nsConn := &tsuki.SpyNSConnector{}

    fsd := tsuki.NewFileServer(store, nsConn)

    token := "history"

    // Send WRITE expect for token
    batch := []string{"1", "2", "3", "4"}
    request := tsuki.NewExpectRequest("write", token, batch...)
    response := httptest.NewRecorder()

    fsd.ServeInner(response, request)

    // WRITE chunks
    request = tsuki.NewPostChunkRequest("1", "chunk1", token)
    response = httptest.NewRecorder()
    fsd.ServeClient(response, request)

    request = tsuki.NewPostChunkRequest("3", "whatisthis", token)
    response = httptest.NewRecorder()
    fsd.ServeClient(response, request)

    // Cancel token
    request = tsuki.NewCancelTokenRequest(token)
    response = httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusOK)

    time.Sleep(10 * time.Millisecond)

    for _, id := range batch {
        tsuki.AssertChunkDoesntExists(t, store, id)
    }

    // Try to WRITE again
    request = tsuki.NewPostChunkRequest("2", "again??", token)
    response = httptest.NewRecorder()
    fsd.ServeClient(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)
}

func TestFS_ChunkPurge(t *testing.T) {
    store := tsuki.NewInMemoryChunkStorage(
        map[string]string {
            "0": "chunk0",
            "1": "isnotchunk1",
    })

    nsConn := &tsuki.SpyNSConnector{}

    fsd := tsuki.NewFileServer(store, nsConn)

    t.Run("purge chunk not in use",
    func (t *testing.T) {
        request := tsuki.NewPurgeRequest("0")
        response := httptest.NewRecorder()

        fsd.ServeInner(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)

        time.Sleep(5 * time.Millisecond)

        tsuki.AssertChunkDoesntExists(t, store, "0")
    })

    t.Run("purge expected by 2 clients chunk",
    func (t *testing.T) {
        chunkId := "1"
        chunkText := store.Index[chunkId]
        token1 := "abc"
        token2 := "xyz"

        fsd.Expect(token1, tsuki.ExpectActionRead, chunkId)
        fsd.Expect(token2, tsuki.ExpectActionRead, chunkId)

        request := tsuki.NewPurgeRequest(chunkId)
        response := httptest.NewRecorder()

        fsd.ServeInner(response, request)

        time.Sleep(20 * time.Millisecond)

        // It doesn't delete it until we fulfill expectation
        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, chunkText)

        request = tsuki.NewGetChunkRequest(chunkId, token1)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

        // Still not deleting it...
        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, chunkId, chunkText)

        request = tsuki.NewGetChunkRequest(chunkId, token2)
        response = httptest.NewRecorder()

        fsd.ServeClient(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)

        time.Sleep(20 * time.Millisecond)

        // Now delete
        tsuki.AssertChunkDoesntExists(t, store, chunkId)
    })

        /*
    t.Run("purge chunk in use",
    func (t *testing.T) {

        store.Mu.RLock()

        request := tsuki.NewPurgeRequest("1")
        response := httptest.NewRecorder()

        fsd.ServeInner(response, request)

        tsuki.AssertStatus(t, response.Code, http.StatusOK)
        tsuki.AssertChunkContents(t, store, "1", "isnotchunk1")

        time.Sleep(5 * time.Millisecond)

        store.Mu.RUnlock()

        time.Sleep(5 * time.Millisecond)

        tsuki.AssertChunkDoesntExists(t, store, "1")
    })
        */
}

func TestFS_Probe(t *testing.T) {
    store := tsuki.NewInMemoryChunkStorage(
        map[string]string {
            "0": "chunk0",
            "1": "isnotchunk1",
    })

    spyConn := &tsuki.SpyNSConnector{}

    heart := &tsuki.Heart{
        Poller: spyConn,
        Sleeper: &tsuki.SpySleeper{},
    }

    fsd := tsuki.NewFileServer(store, spyConn)

    request := tsuki.NewProbeRequest("addr1")
    response := httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusOK)

    var info tsuki.FSProbeInfo

    if err := json.Unmarshal(response.Body.Bytes(), &info); err != nil {
        t.Fatalf("could not parse probe JSON output %q, %v", response.Body.String(), err)
    }

    if info.Available != store.BytesAvailable() {
        t.Errorf("got %d bytes available, want %d", info.Available, store.BytesAvailable())
    }

    if spyConn.GetNSAddr() == "" {
        t.Errorf("expected NS address to be set, but it's not")
    }

    heart.Contract()

    if spyConn.PulseCount != 1 {
        t.Errorf("sent %d heartbeats to NS, want %d", spyConn.PulseCount, 1)
    }

    // Subsequent probes do not change NS
    request = tsuki.NewProbeRequest("addr2")
    response = httptest.NewRecorder()

    fsd.ServeInner(response, request)

    tsuki.AssertStatus(t, response.Code, http.StatusUnauthorized)

    if spyConn.GetNSAddr() != "addr1" {
        t.Errorf("subsequent probes changed NS server")
    }
}

func TestFS_Replicate(t *testing.T) {
    // TODO: untested
}
