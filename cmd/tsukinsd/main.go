package main

import (
  "encoding/json"
  "fmt"
  "github.com/gorilla/mux"
  "log"
  "net/http"
)

type ClientMessage struct {
  Status string `json:"status"`
  Message string `json:"message"`
  Objects []string `json:"objects"`
}

func initTree(w http.ResponseWriter, r *http.Request) {
  t = InitTree(conf.Namenode)
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(&ClientMessage{Status: "OK", Message: "The tree is initialized"})
}

func ls(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  address := r.URL.Query().Get("address")
  list, err := t.LS(address)
  if err == nil {
    json.NewEncoder(w).Encode(&ClientMessage{
      Status: "OK",
      Message: fmt.Sprintf("The content of %s", address),
      Objects: list})
  } else {
    json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
  }
}

func mkdir(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  dirName := r.URL.Query().Get("dir")
  err := t.CreateDirectory(dirName)
  if err == nil {
    json.NewEncoder(w).Encode(&ClientMessage{
      Status: "OK",
      Message: fmt.Sprintf("%d %s directory successfullly created", t.Version, dirName)})
  } else {
    json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
  }
}

func touch(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  address := r.URL.Query().Get("address")
  err := t.CreateFile(address)
  if err == nil {
    json.NewEncoder(w).Encode(&ClientMessage{
      Status: "OK",
      Message: fmt.Sprintf("%d %s file successfullly created", t.Version, address)})
  } else {
    json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
  }
}

func cd(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  address := r.URL.Query().Get("address")
  address, err := t.CD(address)
  if err == nil {
    json.NewEncoder(w).Encode(&ClientMessage{
      Status: "OK",
      Message: fmt.Sprintf("You can change directory to %s", address),
      Objects: []string{address},
    })
  } else {
    json.NewEncoder(w).Encode(&ClientMessage{Status: "ERR", Message: err.Error()})
  }
}

var t *Tree
var conf *Config

func main() {
  var err error
  conf, err = LoadConfig()

  if err != nil {
    log.Fatal(err)
  }

  t = LoadTree(conf.Namenode.TreeGobName)

  r := mux.NewRouter()
  r.HandleFunc("/init", initTree).Methods("GET")
  r.HandleFunc("/ls", ls).Methods("GET")
  r.HandleFunc("/mkdir", mkdir).Methods("GET")
  r.HandleFunc("/touch", touch).Methods("GET")
  r.HandleFunc("/cd", cd).Methods("GET")
  log.Fatal(http.ListenAndServe(":8000", r))
}
