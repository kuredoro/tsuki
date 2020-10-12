package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

const (
    TempNS = "tsuki.ns"
    TempCwd = "tsuki.cwd"
)

const NSCLIENTPORT = ":7070"

var ns string
var cwd string

type ChunkMessage struct {
	ChunkID   string `json:"chunkID"`
	StorageIP string `json:"storageIP"`
}

type ClientMessage struct {
	Status  int            `json:"-"`
	Message string         `json:"message"`
	Objects []string       `json:"objects"`
	Token   string         `json:"token"`
	Chunks  []ChunkMessage `json:"chunks"`
}

type NSClientConnector struct {
	NSAddr string
}

func UnmarshalNSResponse(response *http.Response) (msg *ClientMessage) {
	buf := &bytes.Buffer{}
	io.Copy(buf, response.Body)

	msg = &ClientMessage{Status: response.StatusCode}

	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		log.Printf("warning: could not unmarshal response %q, %v", buf, err)
	}

	return
}

func (conn *NSClientConnector) GetNS(cmd, path string) (*http.Response, error) {
	addr := fmt.Sprintf("http://%s%s/%s?address=%s", conn.NSAddr, NSCLIENTPORT, cmd, path)
	return http.Get(addr)
}

func (conn *NSClientConnector) Ls(path string) ([]string, error) {
	log.Print("ls ", path)

	resp, err := conn.GetNS("ls", path)
	if err != nil {
		log.Printf("error: ls GET error, %v", err)
		return nil, fmt.Errorf("send request: %v", err)
	}

	msg := UnmarshalNSResponse(resp)

	if msg.Status != http.StatusOK {
		return nil, fmt.Errorf(msg.Message)
	}

	log.Printf("Received message: %#v", msg)

	return msg.Objects, nil
}


func saveCwd() {
}

func loadFromTemp(name string) string {
    filename := path.Join(os.TempDir(), name)
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return ""
    }

    file, err := os.Open(filename)
    if err != nil {
        log.Printf("warning: could not open temp file, %v", err)
        return ""
    }
    defer file.Close()

    buf := &bytes.Buffer{}
    io.Copy(buf, file)

    return buf.String()
}

func main() {

    ns := loadFromTemp(TempNS)

	conn := &NSClientConnector{
		NSAddr: ns,
	}
    log.Print(ns)

    cwd = loadFromTemp(TempCwd)
    if cwd == "" {
        cwd = "/"
        saveCwd()
    }

	app := &cli.App{
		Name:  "tsuki",
		Usage: "a CLI interface to tsukiFS distributed file system",
        Commands: []*cli.Command {
            {
                Name: "connect",
                Usage: "Probe and remember name server for future calls",
                Action: func(c *cli.Context) error {
                    conn.NSAddr = c.Args().First()

                    _, err := conn.Ls("/")
                    if err != nil {
                        return fmt.Errorf("%v", err)
                    }

                    file, err := os.Create(path.Join(os.TempDir(), TempNS))
                    if err != nil {
                        fmt.Printf("warning: could not create NS temp file: %v", err)
                        return nil
                    }
                    defer file.Close()

                    fmt.Fprint(file, conn.NSAddr)

                    return nil
                },
            },
            {
                Name: "ls",
                Usage: "List directory",
                Action: func(c *cli.Context) error {

                    path := c.Args().First()
                    if path == "" {
                        path = cwd
                    }

                    objects, err := conn.Ls("/")
                    if err != nil {
                        return fmt.Errorf("%v", err)
                    }

                    for _, obj := range objects {
                        fmt.Println(obj)
                    }

                    return nil
                },
            },
        },
	}

    err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
	}

    /*
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
    */
}
