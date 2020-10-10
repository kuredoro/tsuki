package tsuki_test

import (
	"testing"
	"time"

	"github.com/kureduro/tsuki"
)

func TestHeart(t *testing.T) {
    spyPoller := &tsuki.SpyPoller{}
    spySleeper := &tsuki.SpySleeper{}

    heart := &tsuki.Heart{
        Poller: spyPoller,
        Sleeper: spySleeper,
    }

    const tickCount = 10
    heart.Poll(tickCount)

    if spyPoller.CallCount != tickCount {
        t.Errorf("got %d polls, want %d", spyPoller.CallCount, tickCount)
    }

    if spySleeper.CallCount != tickCount {
        t.Errorf("got %d calls to sleep, want %d", spySleeper.CallCount, tickCount)
    }
}

func TestConfigurableSleeper(t *testing.T) {
    spySleeperTime := &tsuki.SpySleeperTime{}

    sleepFor := 3 * time.Second
    sleeper := &tsuki.ConfigurableSleeper{sleepFor, spySleeperTime.Sleep}
    sleeper.Sleep()

    if spySleeperTime.DurationSlept != sleepFor {
        t.Errorf("slept for %v, but expected to sleep %v", spySleeperTime.DurationSlept, sleepFor)
    }

}

// TODO: tests for HTTPPoller
