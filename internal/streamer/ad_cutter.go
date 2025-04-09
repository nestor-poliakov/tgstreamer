package streamer

import (
	"context"
	"tgstreamer/internal/rpc"
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

func (c *AdCutter) Run(ctx context.Context) {
	go c.processingLoop(ctx)
}

func (c *AdCutter) processingLoop(ctx context.Context) {
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
			close(c.out)
			return
		}
	}
}

func (c *AdCutter) processPiece(piece piece) piece {
	//TODO: cut ad from packets
	return piece
}
