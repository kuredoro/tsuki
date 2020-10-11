package tsuki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
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
//type TokenExpectations map[string]map[string]ExpectAction
type TokenExpectation struct {
    action ExpectAction
    processedChunks map[string]bool
    pendingCount int
    mu sync.RWMutex
}

type ExpectationDB struct {
    index map[string]*TokenExpectation
    mu sync.RWMutex
}

func NewExpectationDB() *ExpectationDB {
    return &ExpectationDB {
        index: make(map[string]*TokenExpectation),
    }
}

func (e *ExpectationDB) Get(token string) *TokenExpectation {
    e.mu.RLock()
    defer e.mu.RUnlock()

    return e.index[token]
}

func (e *ExpectationDB) Set(token string, exp *TokenExpectation) {
    e.mu.Lock()
    defer e.mu.Unlock()

    e.index[token] = exp
}

func (e *ExpectationDB) Remove(token string) {
    e.mu.Lock()
    defer e.mu.Unlock()

    delete(e.index, token)
}

type FSProbeInfo struct {
    Available int
}

type ChunkDB interface {
    Get(id string) (io.Reader, func(), error)
    Create(id string) (io.Writer, func(), error)
    Exists(id string) bool
    // Should be concurrency safe
    Remove(id string) error

    BytesAvailable() int
}

type FileServer struct {
    chunks ChunkDB
    expectations *ExpectationDB
    nsConn NSConnector

    // clientHandler ...also, maybe
    innerHandler http.Handler
}

func NewFileServer(store ChunkDB, nsConn NSConnector) (s *FileServer) {
    s = &FileServer{
        chunks: store,
        expectations: NewExpectationDB(),
        nsConn: nsConn,
    }

    innerRouter := http.NewServeMux()
    innerRouter.Handle("/expect/", http.HandlerFunc(s.ExpectHandler))
    innerRouter.Handle("/cancelToken", http.HandlerFunc(s.CancelTokenHandler))
    innerRouter.Handle("/purge", http.HandlerFunc(s.PurgeHandler))
    innerRouter.Handle("/probe", http.HandlerFunc(s.ProbeHandler))

    s.innerHandler = innerRouter

    return
}

func (s *FileServer) Expect(token string, action ExpectAction, chunks ...string) error {
    // TODO: timeout
    e := s.expectations.Get(token)
    if e != nil {
        return fmt.Errorf("expect group already exists, token=%s", token)
    }

    e = &TokenExpectation {
        action: action,
        processedChunks: make(map[string]bool),
        pendingCount: len(chunks),
    }

    for _, id := range chunks {
        // This if looks kinda crammed and out of context....
        // What if we have more types of expect actions?
        if action == ExpectActionRead && !s.chunks.Exists(id) {
            return ErrChunkNotFound
        }
        e.processedChunks[id] = false
    }

    s.expectations.Set(token, e)

    return nil
}

func (s *FileServer) fullfilExpectation(token, id string) {
    // Expects correct token and id

    e := s.expectations.Get(token)

    e.mu.Lock()
    defer e.mu.Unlock()

    _, chunkExists := e.processedChunks[id]
    if !chunkExists {
        log.Printf("warning: attempt to fullfil expectation for wrong chunk. token=%s, chunk=%s", token, id)
        return
    }

    e.processedChunks[id] = true
    e.pendingCount--

    if e.pendingCount == 0 {
        s.expectations.Remove(token)
    }
}

func (s *FileServer) GetTokenExpectationForChunk(token, id string) ExpectAction {
    e := s.expectations.Get(token)
    if e == nil {
        return ExpectActionNothing
    }

    e.mu.RLock()
    defer e.mu.RUnlock()

    processed, authorized := e.processedChunks[id]
    if !authorized || processed {
        return ExpectActionNothing
    }

    return e.action
}

func (s *FileServer) ServeInner(w http.ResponseWriter, r *http.Request) {
    log.Printf("ServeInner: %s", r.URL)

    s.innerHandler.ServeHTTP(w, r)
}

func (s *FileServer) ExpectHandler(w http.ResponseWriter, r *http.Request) {
    actionStr := r.URL.Query().Get("action")
    action, correct := strToExpectAction[actionStr]

    if !correct {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    token := strings.TrimPrefix(r.URL.Path, "/expect/")

    // TODO: This check may be unneeded when trailing slash is omitted when
    // nothing follows it.
    if token == "" {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    buf := &bytes.Buffer{}
    io.Copy(buf, r.Body)

    var chunks []string
    if err := json.Unmarshal(buf.Bytes(), &chunks); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    err := s.Expect(token, action, chunks...)
    if err != nil {
        w.WriteHeader(http.StatusForbidden)
        fmt.Fprint(w, err)
        return
    }

    log.Printf("%s : %v", r.URL, chunks)

    w.WriteHeader(http.StatusOK)
}

func (s *FileServer) CancelTokenHandler(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")

    if token == "" {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    e := s.expectations.Get(token)
    if e == nil {
        w.WriteHeader(http.StatusOK)
        return
    }

    e.mu.Lock()
    defer e.mu.Unlock()


    toUndo := make([]string, 0, len(e.processedChunks) - e.pendingCount)
    for k, v := range e.processedChunks {
        if v {
            toUndo = append(toUndo, k)
        }
    }

    if e.action == ExpectActionWrite && len(toUndo) != 0 {
        // Remove blocks
        for _, id := range toUndo {
            // TODO: Start go routines
            err := s.chunks.Remove(id)
            if err != nil {
                log.Fatal(err)
            }
        }
    }

    e.action = ExpectActionNothing

    s.expectations.Remove(token)

    w.WriteHeader(http.StatusOK)
}

func (s *FileServer) PurgeHandler(w http.ResponseWriter, r *http.Request) {
    buf := &bytes.Buffer{}
    io.Copy(buf, r.Body)

    var chunks []string
    if err := json.Unmarshal(buf.Bytes(), &chunks); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    for _, id := range chunks {
        go s.chunks.Remove(id)
    }

    w.WriteHeader(http.StatusOK)
}

func (s *FileServer) ProbeHandler(w http.ResponseWriter, r *http.Request) {
    if !s.nsConn.IsNS(r.RemoteAddr) {
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    s.nsConn.SetNSAddr(r.RemoteAddr)

    info := s.GenerateProbeInfo()

    probeBytes, err := json.Marshal(info)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fmt.Fprint(w, string(probeBytes))
    w.WriteHeader(http.StatusOK)
}

func (s *FileServer) GenerateProbeInfo() *FSProbeInfo {
    return &FSProbeInfo {
        Available: s.chunks.BytesAvailable(),
    }
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
    log.Printf("Chunk READ request: id=%s, token=%s", id, token)

    if s.GetTokenExpectationForChunk(token, id) == ExpectActionNothing {
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
    log.Printf("Chunk WRITE request: id=%s, token=%s", id, token)

    if s.GetTokenExpectationForChunk(token, id) == ExpectActionNothing {
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

    s.nsConn.ReceivedChunk(id)
    w.WriteHeader(http.StatusOK)
}
