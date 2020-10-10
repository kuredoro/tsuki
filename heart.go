package tsuki

import (
	"log"
	"net/http"
	"time"
)

type Poller interface {
    Poll()
}

type Sleeper interface {
    Sleep()
}

// HTTPPoller will poll the specified URL link. The link should contain
// http:// at the beginning.
type HTTPPoller struct {
    address string
}

func (p *HTTPPoller) Poll() {
    _, err := http.Get(p.address)

    if err != nil {
        log.Printf("warning: could sent hertbeat to NS")
    }
}

type ConfigurableSleeper struct {
    Duration time.Duration
    SleepFunc func(time.Duration)
}

func (s *ConfigurableSleeper) Sleep() {
    s.SleepFunc(s.Duration)
}

type Heart struct {
    Poller Poller
    Sleeper Sleeper
}

func NewHeart(pollAddr string, sleepFor time.Duration) *Heart {
    return &Heart{
        Poller: &HTTPPoller{ address: pollAddr },
        Sleeper: &ConfigurableSleeper{ 
            Duration: sleepFor, 
            SleepFunc: time.Sleep,
        },
    }
}

// Poll will make count consequent polls with calls to sleeper in-between.
// Set count to -1, to poll indefinetely.
func (h *Heart) Poll(count int) {
    if count != -1 {
        for i := 0; i < count; i++ {
            h.contract()
        }

        return
    }

    for {
        h.contract()
    }
}

func (h *Heart) contract() {
    h.Poller.Poll()
    h.Sleeper.Sleep()
}
