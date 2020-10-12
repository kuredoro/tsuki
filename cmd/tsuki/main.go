package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

type ChunkMessage struct {
	ChunkID   string `json:"chunkID"`
	StorageIP string `json:"storageIP"`
}

type ClientMessage struct {
	Status  int         `json:"-"`
	Message string         `json:"message"`
	Objects []string       `json:"objects"`
	Token   string         `json:"token"`
	Chunks  []ChunkMessage `json:"chunks"`
}

type NSClientConnector struct {
	nsAddr string
}

func UnmarshalNSResponse(response *http.Response) (msg *ClientMessage) {
	buf := &bytes.Buffer{}
	io.Copy(buf, response.Body)

    msg = &ClientMessage{ Status: response.StatusCode }

    if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
        log.Printf("warning: could not unmarshal response %q, %v", buf, err)
    }
    
    return
}

func (conn *NSClientConnector) Ls(path string) ([]string, error) {
	log.Print("ls ", path)

	addr := fmt.Sprintf("http://%s/%s?address=%s", conn.nsAddr, "ls", path)
	resp, err := http.Get(addr)
	if err != nil {
		log.Printf("error: ls GET error, %v", err)
		return nil, fmt.Errorf("could not send request, %v", err)
	}

    msg := UnmarshalNSResponse(resp)

    if msg.Status != http.StatusOK {
        return nil, fmt.Errorf(msg.Message)
    }

	log.Printf("Received message: %#v", msg)

	return msg.Objects, nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Printf("Usage: tsuki hostaddr:port\n")
		return
	}

	conn := &NSClientConnector{
		nsAddr: os.Args[1],
	}

    curPath := "/"

	_, err := conn.Ls(curPath)
	if err != nil {
		fmt.Println("could not connect to NS, ", err)
		return
	}

    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Printf("%s $ ", curPath)

        scanner.Scan()

        cmd := strings.Split(scanner.Text(), " ")

        if len(cmd) == 0 {
            continue
        }

        switch cmd[0] {
        case "ls":
            lsDir := curPath
            if len(cmd) == 2 {
                lsDir = path.Clean(cmd[1])
            }

            objects, err := conn.Ls(lsDir)
            if err != nil {
                fmt.Printf("error: %v\n", err)
                continue
            }

            fmt.Println(strings.Join(objects, "\n"))
        case "exit", "quit":
            return
        default:
            fmt.Printf("error: unknown command %q\n", cmd[0])
        }
    }
}
