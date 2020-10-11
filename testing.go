package tsuki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type InMemoryChunkStorage struct {
    Index map[string]string
    mu sync.RWMutex
    accessCount sync.WaitGroup
    atLeastOneCall chan struct{}
    callsPerformed int
}

func NewInMemoryChunkStorage(index map[string]string) *InMemoryChunkStorage {
    return &InMemoryChunkStorage {
        Index: index,
        atLeastOneCall: make(chan struct{}),
    }
}

func (s *InMemoryChunkStorage) breakBarrier() {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.callsPerformed == 0 {
        close(s.atLeastOneCall)
        s.callsPerformed++
    }
}

func (s *InMemoryChunkStorage) Get(id string) (io.Reader, func(), error) {
    s.breakBarrier()

    if !s.Exists(id) {
        return nil, func(){}, ErrChunkNotFound
    }

    s.accessCount.Add(1)

    s.mu.RLock()
    defer s.mu.RUnlock()

    chunk := s.Index[id]

    buf := bytes.NewBufferString(chunk)

    closeFunc := func() {
        s.accessCount.Done()
    }

    return buf, closeFunc, nil
}

func (s *InMemoryChunkStorage) Create(id string) (io.Writer, func(), error) {
    s.breakBarrier()

    if s.Exists(id) {
        return nil, func(){}, ErrChunkExists
    }

    s.accessCount.Add(1)

    s.mu.Lock()
    defer s.mu.Unlock()

    s.Index[id] = ""

    buf := &bytes.Buffer{}

    writeChunk := func() {
        str := &strings.Builder{}
        io.Copy(str, buf)
        s.Index[id] = str.String()

        s.accessCount.Done()
    }

    return buf, writeChunk, nil
}

func (s *InMemoryChunkStorage) Exists(id string) (exists bool) {
    s.accessCount.Add(1)
    defer s.accessCount.Done()

    s.mu.RLock()
    defer s.mu.RUnlock()

    _, exists = s.Index[id]
    return
}

func (s *InMemoryChunkStorage) Remove(id string) error {
    s.breakBarrier()

    s.accessCount.Add(1)
    defer s.accessCount.Done()

    s.mu.Lock()
    defer s.mu.Unlock()

    delete(s.Index, id)
    return nil
}

func (s *InMemoryChunkStorage) Wait() {
    <-s.atLeastOneCall
    s.accessCount.Wait()
}

func NewGetChunkRequest(id, token string) *http.Request {
    req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/chunks/%s?token=%s", id, token), nil)
    return req
}

func NewPostChunkRequest(id, content, token string) *http.Request {
    buf := bytes.NewBufferString(content)
    request, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/chunks/%s?token=%s", id, token), buf)
    return request
}

func NewExpectRequest(method, token string, chunks []string) *http.Request {
    b, _ := json.Marshal(chunks)
    url := fmt.Sprintf("/expect/%s?token=%s", method, token)
    req, _ := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(b))
    return req
}

func NewCancelTokenRequest(token string) *http.Request {
    url := fmt.Sprintf("/cancelToken?token=%s", token)
    req, _ := http.NewRequest(http.MethodPost, url, nil)
    return req
}

func NewPurgeRequest(chunks ...string) *http.Request {
    b, _ := json.Marshal(chunks)
    url := "/purge"
    req, _ := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(b))
    return req
}

func AssertChunkContents(t *testing.T, chunks ChunkDB, id, want string) {
    t.Helper()

    if !chunks.Exists(id) {
        t.Fatalf("expected chunk %v to exist, but it doesn't", id)
    }

    chunk, closeChunk, _ := chunks.Get(id)
    defer closeChunk()

    got := &strings.Builder{}
    _, _ = io.Copy(got, chunk)


    if got.String() != want {
        t.Errorf("got chunk contents %q, want %q", got, want)
    }
}

func AssertChunkDoesntExists(t *testing.T, chunks ChunkDB, id string) {
    t.Helper()

    if chunks.Exists(id) {
        t.Errorf("chunk %v exists, but it shouldn't", id)
    }
}

func AssertStatus(t *testing.T, got, want int) {
    t.Helper()
    if got != want {
        t.Errorf("incorrect response code, got %d, want %d", got, want)
    }
}

func AssertResponseBody(t *testing.T, got, want string) {
    t.Helper()
    if got != want {
        t.Errorf("wrong response body, got %q, want %q", got, want)
    }
}

func AssertReceivedChunkCalls(t *testing.T, nsConn *SpyNSConnector, ids ...string) {
    t.Helper()

    if !reflect.DeepEqual(nsConn.receivedChunks, ids) {
        t.Errorf("incorrect calls to ns/receivedChunk, got %#v, %#v", nsConn.receivedChunks, ids)
    }
}


type SpyPoller struct {
    CallCount int
}

func (p *SpyPoller) Poll() {
    p.CallCount++
}


type SpySleeper struct {
    CallCount int
}

func (s *SpySleeper) Sleep() {
    s.CallCount++
}


type SpySleeperTime struct {
    DurationSlept time.Duration
}

func (s *SpySleeperTime) Sleep(duration time.Duration) {
    s.DurationSlept = duration
}

