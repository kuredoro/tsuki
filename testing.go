package tsuki

import (
    "fmt"
    "net/http"
    "testing"
)

func NewGetChunkRequest(id string) *http.Request {
    req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/chunks/%v", id), nil)
    return req
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

