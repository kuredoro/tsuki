package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type FSStatus int

const (
	LIVE           FSStatus = 2
	PARTIALLY_DEAD FSStatus = 1
	DEAD           FSStatus = 0
)

type FileServerInfo struct {
	mu        sync.Mutex
	Host      string
	Port      int
	Alive     bool
	Status    FSStatus
	NextAlive int
	LastPulse time.Time
	ID        int
}

type PoolInfo struct {
	mu             sync.Mutex
	StorageNodes   []*FileServerInfo
	SoftPulseQueue chan int
	HardPulseQueue chan int
	http.Handler
	Next  int
	Alive int
}

func InitFServers(conf *Config) *PoolInfo {
	storage := PoolInfo{
		SoftPulseQueue: make(chan int, 1),
		HardPulseQueue: make(chan int, 1),
	}
	for i, storageNode := range conf.Storage {
		storage.StorageNodes = append(storage.StorageNodes,
			&FileServerInfo{
				Host:      storageNode.Host,
				Port:      storageNode.Port,
				Alive:     false,
				NextAlive: (i + 1) % len(conf.Storage),
				ID:        i,
			})
	}

	return &storage
}

func (s *PoolInfo) Select() *FileServerInfo {
	next := s.StorageNodes[s.Next]

	if !next.Alive {
		next = s.StorageNodes[next.NextAlive]
	}

	s.Next = next.NextAlive

	return next
}

func (s *PoolInfo) SelectSeveralExcept(except []string, num int) []*FileServerInfo {
	exceptMap := map[string]int{}
	for _, host := range except {
		exceptMap[host] = 1
	}
	//if s.Alive-len(except) < num {
	//	num = s.Alive - len(except)
	//}

	selected := []*FileServerInfo{}

	next := s.StorageNodes[s.Next]
	start := next
	for i := 0; i < num; {
		if !next.Alive || exceptMap[next.Host] == 1 {
		} else {
			selected = append(selected, next)
			i++
		}
		next = s.StorageNodes[next.NextAlive]

		if &next == &start {
			break
		}
	}

	return selected
}

func (s *PoolInfo) setNewAliveDead(nowDeadID int) {
	s.setNewAlive(s.StorageNodes[nowDeadID].NextAlive, nowDeadID)
}

func (s *PoolInfo) setNewAlive(newAliveID int, cur int) {
	setOne := false

	s.mu.Lock()
	for i := cur; i >= 0; i-- {
		node := s.StorageNodes[i]
		node.NextAlive = newAliveID
		if node.Alive {
			setOne = true
			break
		}
	}
	s.mu.Unlock()

	if !setOne {
		if cur != len(s.StorageNodes)-1 {
			s.setNewAlive(newAliveID, len(s.StorageNodes)-1)
		} else {
			// no alive node; die
			// panic("no alive node")
		}
	}
}

func SendChunksToFS(inversed map[string][]string, token string) {
	for host, chunks := range inversed {
		jsonStr, _ := json.Marshal(chunks)
		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/expect/write?token=%s", host, token), bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			// cancel token
			return
		}

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
		resp.Body.Close()

		// if any error here die
	}
}

func Replicate(chunk *Chunk, sender string, receiver *FileServerInfo) {
	client := &http.Client{}
	json := []byte(fmt.Sprintf("[%s]", chunk.ChunkID))

	token := generateToken()

	req, _ := http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:%d/expect?token=%s&action=write", receiver.Host, conf.Namenode.StoragePrivatePort, token),
		bytes.NewBuffer(json))
	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		// cancel token (cancelToken)
		return
	}

	req, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:%d/replicate?token=%s&ip=%s",
			sender, conf.Namenode.StoragePrivatePort, token, receiver.Host+":"+strconv.Itoa(receiver.Port)),
		bytes.NewBuffer(json))
	req.Header.Set("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		// cancel token (cancelToken)
		return
	}
}

func (s *PoolInfo) ChangeStatus(id int, status FSStatus) {
	node := s.StorageNodes[id]

	node.mu.Lock()
	prevState := node.Alive
	node.Status = status
	node.Alive = status == LIVE
	node.mu.Unlock()

	changedState := node.Alive && !(prevState && node.Alive)

	if node.Alive {
		num := len(s.StorageNodes)
		s.setNewAlive(id, ((id-1)%num+num)%num)

		s.mu.Lock()
		s.Alive += 1
		s.mu.Unlock()

	} else {
		if changedState {
			s.mu.Lock()
			s.Alive -= 1
			s.mu.Unlock()
		}
		s.setNewAliveDead(id)
	}
}

func (s *PoolInfo) PurgeChunks(id int, chunks []string) {
	fs := s.StorageNodes[id]

	jsonChunks, _ :=json.Marshal(chunks)

	client := &http.Client{}

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%s:%d/purge", fs.Host, conf.Namenode.StoragePrivatePort),
		bytes.NewBuffer(jsonChunks))

	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		// todo: add to queue
		return
	}
}

func (s *PoolInfo) IsDead(id int, soft bool) bool {
	return soft && s.StorageNodes[id].GetStatus() == PARTIALLY_DEAD || !soft && s.StorageNodes[id].GetStatus() == DEAD
}

func (fs *FileServerInfo) GetStatus() FSStatus {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.Status
}

func (s *PoolInfo) NodeIsDead(id int) {

}
