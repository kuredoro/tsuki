package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

type PendingUploads struct {
	tokens map[string]string
	fss    []*FileServerInfo
}

func (s *PoolInfo) HeartbeatManager(soft bool) {
	var period time.Duration
	var deathStatus FSStatus
	var liveStatus FSStatus
	var queue chan int
	if soft {
		period = conf.Namenode.SoftDeathTime * time.Second
		deathStatus = PARTIALLY_DEAD
		liveStatus = LIVE
		queue = s.SoftPulseQueue
	} else {
		period = conf.Namenode.HardDeathTime * time.Second
		deathStatus = DEAD
		liveStatus = PARTIALLY_DEAD
		queue = s.HardPulseQueue
	}

	nextDead := -1

	var deathTime = period


	for {
		select {
		case peerId := <-queue:
			if s.IsDead(peerId, soft) {
				// wow, it is alive now! do some stuff to resurrect it
				log.Printf("%d became live now!; partially: %v", peerId, soft)
				s.ChangeStatus(peerId, liveStatus)
				nextDead, deathTime = s.GetFSWithOldestPulse(soft)
				//log.Printf("%v %v %d %v", soft, deathTime, nextDead, s.StorageNodes[peerId].Status)
				log.Printf("active %v deathTime %v nextDead %v status %d peerid %v", s.Alive, deathTime, nextDead, s.StorageNodes[peerId].Status, peerId)
			} else if peerId == nextDead || nextDead == -1 {
				nextDead, deathTime = s.GetFSWithOldestPulse(soft)
				//log.Printf("soft %v deathTime %v nextDead %v status %d peerid %v", soft, deathTime, nextDead, s.StorageNodes[peerId].Status, peerId)
			} else {
				nextDead, deathTime = s.GetFSWithOldestPulse(soft)
			}
		case <-time.After(deathTime):
			log.Printf("sos %v\n", soft)
			if nextDead == -1 {
				continue
			}

			// Mark as dead
			s.ChangeStatus(nextDead, deathStatus)

			log.Printf("%d is dead now; partially: %v", nextDead, soft)
			nextDead, _ = s.GetFSWithOldestPulse(soft)
			deathTime = period
		}
	}
}

func (s *PoolInfo) GetFSWithOldestPulse(soft bool) (int, time.Duration) {
	// For loop in the array and pick one with the oldest pulse time
	// or whatever...
	var oldest = -1
	var period time.Duration

	if soft {
		period = conf.Namenode.SoftDeathTime * time.Second
	} else {
		period = conf.Namenode.HardDeathTime * time.Second
	}

	oldestDuration := time.Duration(0)
	for i, fs := range s.StorageNodes {
		if since := time.Since(fs.LastPulse); since > oldestDuration && (soft && fs.Status == LIVE || !soft && fs.Status != DEAD) {
			oldestDuration = since
			oldest = i
		}
	}

	return oldest, period - oldestDuration
}

func ExpectFromFS(chunk *Chunk, sender string, receiver *FileServerInfo) {
	token := generateToken()
	_, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:%d/write?token=%s&chunkID=%s", receiver.Host, receiver.Port, token, chunk.ChunkID),
		nil)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:%d/expect/write?token=%s&chunkID=%s", receiver.Host, receiver.Port, token, chunk.ChunkID),
		nil)

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		// cancel token
		return
	}
}

func pulse(w http.ResponseWriter, r *http.Request) {
	//remoteHost := strings.Split(r.RemoteAddr, ":")[0]
	remoteHost := r.Header.Get("addr")
	for _, fs := range storages.StorageNodes {
		if fs.Host == remoteHost {
			log.Printf("Received heart beat from: %s", remoteHost)
			fs.LastPulse = time.Now()
			storages.HardPulseQueue <- fs.ID
			storages.SoftPulseQueue <- fs.ID
			break
		}
	}
	log.Printf("Received heart beat from unknown host: %s", remoteHost)
	w.WriteHeader(http.StatusOK)

}

func confirmChunk(w http.ResponseWriter, r *http.Request) {
	// chunk is ready at r.RemoteAddr
	// we can set it as ready on remote addr and start sending to other servers
	chunkID := r.URL.Query().Get("chunkID")
	remoteAddr := r.Header.Get("addr")
	//remoteAddr := strings.Split(r.RemoteAddr, ":")[0]
	log.Printf("Got ready chunk %s from %s", chunkID, remoteAddr)

	chunk, ok := ct.Table[chunkID]
	if !ok {
		// here send request to remove chunk since it does not exist on the ns
		log.Printf("Chunk %s not found; skipping", chunkID)
		return
	}

	_, ok = chunk.Statuses[remoteAddr]

	if !ok {
		log.Printf("Got chunk %s from %s but it should not be there...", chunkID, remoteAddr)
		return
	}

	chunk.Statuses[remoteAddr] = OK
	file, ok := t.GetNodeByAddress(chunk.File)
	if !ok {
		log.Printf("File %s not found; skipping", chunk.File)
		return
	}
	delete(file.Pending, chunkID)

	chunk.ReadyReplicas += 1
	remainingReplicas := conf.Namenode.Replicas - chunk.AllReplicas

	senders := []string{}
	for fs, status := range chunk.Statuses {
		if status == OK {
			senders = append(senders, fs)
			if len(senders) == remainingReplicas {
				break
			}
		}
	}

	remainingReplicas = Min(remainingReplicas, len(senders))
	receivers := storages.SelectSeveralExcept(senders, remainingReplicas)

	if len(receivers) == 0 && chunk.AllReplicas < conf.Namenode.Replicas {
		log.Printf("Chunk %s cannot be replicated more, there is no free fs left", chunkID)
		// todo: add to some queue that is subscribed to events when some fs are up
	}

	chunk.AllReplicas += len(receivers)

	for i, receiver := range receivers {
		go ExpectFromFS(chunk, senders[i], receiver)
		log.Printf("Sending chunk %s from %s to %v", chunkID, senders[i], receiver)
		chunk.AddFSToChunk(receiver)
	}
}

func printTree(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	t.PrintTreeStruct()
}

func StartPrivateServer() {
	r := mux.NewRouter()
	r.HandleFunc("/pulse", pulse).Methods("GET", "POST")
	r.HandleFunc("/confirm/chunk", confirmChunk).Methods("GET")
	r.HandleFunc("/print", printTree).Methods("GET")

	http.ListenAndServe(fmt.Sprintf("%s:%d", conf.Namenode.Host, conf.Namenode.PrivatePort), r)
}

