package streamer

import (
	"context"
	"sync"
	"tgstreamer/internal/rpc"
	"tgstreamer/lib/log"
)

type AdCutter struct {
	sb  *rpc.SponsorBlockClient
	in  chan piece
	out chan piece
}

func NewAdCutter(in chan piece, out chan piece) *AdCutter {
	return &AdCutter{
		in:  in,
		out: out,
	}
}

func (c *AdCutter) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx = log.With(ctx, "worker", "ad_cutter")
	wg.Add(1)
	go c.processingLoop(ctx, wg)
}

func (c *AdCutter) processingLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(c.out)
	for {
		select {
		case piece, ok := <-c.in:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case c.out <- c.processPiece(piece):
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *AdCutter) processPiece(piece piece) piece {
	//TODO: cut ad from packets
	return piece
}
