package util

import (
	"log"
	"time"
)

type PerfTimer struct {
	startTime, lastSplitTime time.Time
}

func NewPerfTimer() *PerfTimer {
	return &PerfTimer{
		startTime:     time.Now(),
		lastSplitTime: time.Now(),
	}
}

func (pt *PerfTimer) GetSplit() (total, split time.Duration) {
	now := time.Now()
	total = now.Sub(pt.startTime)
	split = now.Sub(pt.lastSplitTime)
	pt.lastSplitTime = now
	return
}

func (pt *PerfTimer) LogSplit(eventName string) {
	total, split := pt.GetSplit()
	log.Printf("%s: %fs split, %fs since start\n", eventName, split.Seconds(), total.Seconds())
}
