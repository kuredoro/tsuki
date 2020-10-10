package tsuki

import (
	"net/http"
	"strings"
    "io"
)

type chunkError string

func (c chunkError) Error() string { return string(c) }

const (
    ErrChunkExists = chunkError("chunk already exists")
    ErrChunkNotFound = chunkError("chunk does not exists")
)

type ChunkDB interface {
    Get(id string) (io.Reader, func(), error)
    Create(id string) (io.Writer, func(), error)
    Exists(id string) bool
}

type FileServer struct {
    chunks ChunkDB
}

func NewFileServer(store ChunkDB) *FileServer {
    return &FileServer{
        chunks: store,
    }
}

func (cs *FileServer) ServerClient(w http.ResponseWriter, r *http.Request) {
    chunkId := strings.TrimPrefix(r.URL.Path, "/chunks/")
    
    switch r.Method {
    case http.MethodGet:
        cs.SendChunk(w, chunkId)
    case http.MethodPost:
        cs.DownloadChunk(w, r, chunkId)
    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func (cs *FileServer) SendChunk(w http.ResponseWriter, id string) {
    chunk, closeChunk, err := cs.chunks.Get(id)
    defer closeChunk()

    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    io.Copy(w, chunk)
    return
}

func (cs *FileServer) DownloadChunk(w http.ResponseWriter, r *http.Request, id string) {
    chunk, finishChunk, err := cs.chunks.Create(id)
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
