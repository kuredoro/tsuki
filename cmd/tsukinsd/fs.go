package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
	Available int64
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
	i := 0
	for _, storageNode := range conf.Storage {
		available, ok := ProbeFServer(storageNode.Host, conf.Namenode.StoragePrivatePort)
		if !ok {
			continue
		}

		storage.StorageNodes = append(storage.StorageNodes,
			&FileServerInfo{
				Host:      storageNode.Host,
				Port:      conf.Namenode.StoragePrivatePort,
				Alive:     true,
				Status:    LIVE,
				NextAlive: (i + 1) % len(conf.Storage),
				ID:        i,
				Available: available,
				LastPulse: time.Now(),
			})
		i++
	}

	if len(storage.StorageNodes) < conf.Namenode.Replicas {
		log.Fatal("Not enough servers. Please add more and restart or reduce the number of replicas (number of replicas <= number of FSs)")
	} else {
		storage.StorageNodes[len(storage.StorageNodes)-1].NextAlive = 0
	}

	return &storage
}

func ProbeFServer(host string, port int) (int64, bool) {
	log.Printf("Probing %s:%d", host, port)

	//req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/expect/write?token=%s", host, port), nil)

	// for testing
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/probe?host=%s", "localhost", port, host), nil)

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		log.Printf("Probing %s:%d failed", host, port)
		return 0, false
	}
	body, _ := ioutil.ReadAll(resp.Body)

	res := &struct {
		Available int64 `json:"available"`
	}{}
	err = json.Unmarshal(body, &res)

	if err != nil {
		log.Printf("Probing %s:%d failed", host, port)
		return 0, false
	}

	log.Printf("Probing %s:%d is successful; available: %d", host, port, res.Available)
	return res.Available, true
}

func (s *PoolInfo) Select() *FileServerInfo {
	next := s.StorageNodes[s.Next]

	if !next.Alive {
		next = s.StorageNodes[next.NextAlive]
	}

	s.Next = next.NextAlive

	return next
}

func (s *PoolInfo) SelectSeveralExcept(exceptMap map[string]*FileServerInfo, num int) []*FileServerInfo {
	//if s.Alive-len(except) < num {
	//	num = s.Alive - len(except)
	//}

	selected := []*FileServerInfo{}

	next := s.StorageNodes[s.Next]
	start := next
	for i := 0; i < num; {
		if !next.Alive || exceptMap[next.Host] != nil {
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

func (s *PoolInfo) SelectSeveralExceptArr(except []string, num int) []*FileServerInfo {
	exceptMap := map[string]*FileServerInfo{}
	for _, host := range except {
		exceptMap[host] = &FileServerInfo{}
	}

	return s.SelectSeveralExcept(exceptMap, num)
}

func (s *PoolInfo) SelectAmong(among map[string]*FileServerInfo) (*FileServerInfo, error) {
	next := s.Next
	var chosen *FileServerInfo

	serNum := len(s.StorageNodes)
	diff := 3 * serNum

	for _, fs := range among {
		if fs.Alive && Abs(fs.ID+serNum-next) < diff {
			next := fs.ID
			chosen = fs
			diff = Abs(fs.ID - next)
		}
	}

	if chosen == nil {
		return nil, fmt.Errorf("no available server to select")
	} else {
		s.Next = chosen.NextAlive
		return chosen, nil
	}
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

func ExpectChunksFromClient(inversed map[string][]string, token string) {
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

	ct.InvertedTable[receiver.Host] = append(ct.InvertedTable[receiver.Host], chunk)

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
			sender, conf.Namenode.StoragePrivatePort, token, fmt.Sprintf("%s:%d", receiver.Host, receiver.Port)),
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

	if status == DEAD {
		// start replication process
		s.FSIsDown(node)
	}
}

func (s *PoolInfo) FSIsDown(node *FileServerInfo) {
	log.Printf("OMG, %s is down", node.Host)
	chunks, ok := ct.InvertedTable[node.Host]
	if !ok {
		// nothing to do; no chunks on server
		return
	}

	for _, chunk := range chunks {
		switch chunk.Status {
		case PENDING:
			chunk.Status = DOWN
			log.Fatal("Impossible to replicate. The file is dead now.")
		case OBSOLETE, DOWN:
			// nothing to do
			continue
		}

		sender, _ := s.SelectAmong(chunk.FServers)
		newFS := s.SelectSeveralExcept(chunk.FServers, 1)

		if len(newFS) == 0 {
			// very bad, no new server
			// todo: put it to a queue
			return
		}
		delete(chunk.FServers, node.Host)
		chunk.ReadyReplicas -= 1
		chunk.AllReplicas -= 1

		chunk.AddFSToChunk(newFS[0])

		log.Printf("OMG, %s is down; replicating %s from %s to %s", node.Host, chunk.ChunkID, sender.Host, newFS[0].Host)
		go Replicate(chunk, sender.Host, newFS[0])
	}

}

func (s *PoolInfo) PurgeChunks(id int, chunks []string) {
	fs := s.StorageNodes[id]

	jsonChunks, _ := json.Marshal(chunks)

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
