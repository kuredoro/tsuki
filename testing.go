package tsuki

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
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

