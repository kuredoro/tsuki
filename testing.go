package tsuki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

type InMemoryChunkStorage struct {
    Index map[string]string
}

func (s *InMemoryChunkStorage) Get(id string) (io.Reader, func(), error) {
    chunk, exists := s.Index[id]

    if !exists {
        return nil, func(){}, ErrChunkNotFound
    }

    buf := bytes.NewBufferString(chunk)

    return buf, func(){}, nil
}

func (s *InMemoryChunkStorage) Create(id string) (io.Writer, func(), error) {
    if s.Exists(id) {
        return nil, func(){}, ErrChunkExists
    }

    s.Index[id] = ""

    buf := &bytes.Buffer{}

    writeChunk := func() {
        str := &strings.Builder{}
        io.Copy(str, buf)
        s.Index[id] = str.String()
    }

    return buf, writeChunk, nil
}

func (s *InMemoryChunkStorage) Exists(id string) (exists bool) {
    _, exists = s.Index[id]
    return
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

