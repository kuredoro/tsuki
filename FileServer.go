package tsuki

import (
	"net/http"
	"strings"
    "io"
)

type ChunkError string

func (c ChunkError) Error() string { return string(c) }

const (
    ErrChunkExists = ChunkError("chunk already exists")
    ErrChunkNotFound = ChunkError("chunk does not exists")
)

type ExpectAction int

const (
    ExpectActionNothing = ExpectAction(0)
    ExpectActionRead = ExpectAction(1)
    ExpectActionWrite = ExpectAction(2)
)

var strToExpectAction = map[string]ExpectAction {
    "read": ExpectActionRead,
    "write": ExpectActionWrite,
}

// token -> chunkId -> ExpectAction
// Putting token first makes it more cache-friendly, since there much less
// tokens than chunks.
type TokenExpectations map[string]map[string]ExpectAction

type ChunkDB interface {
    Get(id string) (io.Reader, func(), error)
    Create(id string) (io.Writer, func(), error)
    Exists(id string) bool
}

type FileServer struct {
    chunks ChunkDB
    expectations TokenExpectations
}

func NewFileServer(store ChunkDB) *FileServer {
    return &FileServer{
        chunks: store,
        expectations: make(TokenExpectations),
    }
}

func (s *FileServer) Expect(action ExpectAction, id, token string) {
    // TODO: timeout
    _, exists := s.expectations[token]
    if !exists {
        s.expectations[token] = make(map[string]ExpectAction)
    }

    s.expectations[token][id] = action
}

func (s *FileServer) fullfilExpectation(token, id string) {
    // Expects correct token and id

    delete(s.expectations[token], id)

    if len(s.expectations[token]) == 0 {
        delete(s.expectations, token)
    }
}

func (s *FileServer) getTokenExpectationForChunk(token, id string) ExpectAction {
    _, exists := s.expectations[token]
    if !exists {
        return ExpectActionNothing
    }

    action, exists := s.expectations[token][id]
    if !exists {
        return ExpectActionNothing
    }

    return action
}

func (cs *FileServer) ServeClient(w http.ResponseWriter, r *http.Request) {
    chunkId := strings.TrimPrefix(r.URL.Path, "/chunks/")
    token := r.URL.Query().Get("token")
    
    switch r.Method {
    case http.MethodGet:
        cs.SendChunk(w, r, chunkId, token)
    case http.MethodPost:
        cs.ReceiveChunk(w, r, chunkId, token)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (s *FileServer) SendChunk(w http.ResponseWriter, r *http.Request, id, token string) {

    if s.getTokenExpectationForChunk(token, id) == ExpectActionNothing {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
    defer s.fullfilExpectation(token, id)

    chunk, closeChunk, err := s.chunks.Get(id)
    defer closeChunk()

    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    io.Copy(w, chunk)
    return
}

func (s *FileServer) ReceiveChunk(w http.ResponseWriter, r *http.Request, id, token string) {

    if s.getTokenExpectationForChunk(token, id) == ExpectActionNothing {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }
    defer s.fullfilExpectation(token, id)

    chunk, finishChunk, err := s.chunks.Create(id)
    defer finishChunk()

    if err == ErrChunkExists {
        w.WriteHeader(http.StatusForbidden)
        return
    }

    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    io.Copy(chunk, r.Body)

    w.WriteHeader(http.StatusOK)
}
