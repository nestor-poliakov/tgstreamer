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
		//pktTimeDiff += time.Millisecond * 50
		if wallTimeDiff < pktTimeDiff && time.Since(l.start)-p.Time < time.Second/10 {
			time.Sleep(pktTimeDiff - wallTimeDiff)
		}
		l.pktTimeStart = p.Time
		l.wallTimeStart = time.Now()
	}
}
