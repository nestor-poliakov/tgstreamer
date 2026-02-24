package streamer

import (
	"time"

	"github.com/nestor-poliakov/joy5/av"
)

type rateLimiter struct {
	pktTimeStart  time.Duration
	wallTimeStart time.Time
	start         time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		wallTimeStart: time.Now(),
		start:         time.Now().Add(-time.Second * 1),
	}
}

func (l *rateLimiter) Limit(p av.Packet) {
	if pktTimeDiff := p.Time - l.pktTimeStart; pktTimeDiff > 0 {
		wallTimeDiff := time.Now().Sub(l.wallTimeStart)
		if wallTimeDiff < pktTimeDiff && time.Since(l.start)-p.Time < time.Second/10 {
			time.Sleep(pktTimeDiff - wallTimeDiff)
		}
		l.pktTimeStart = p.Time
	}
}

// Mark resets the wall-clock reference to now. Call it immediately after the
// packet has been written so that write latency is included in the timing
// budget of the next packet, preventing drift.
func (l *rateLimiter) Mark() {
	l.wallTimeStart = time.Now()
}
