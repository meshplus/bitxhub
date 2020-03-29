package p2p

import (
	"context"
	"fmt"
	"io"
	"time"

	ggio "github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	net "github.com/meshplus/bitxhub/pkg/network"
	"github.com/meshplus/bitxhub/pkg/network/proto"
)

// handle newly connected stream
func (p2p *P2P) handleNewStream(s network.Stream) {
	if err := s.SetReadDeadline(time.Time{}); err != nil {
		p2p.logger.WithField("error", err).Error("Set stream read deadline")
		return
	}

	reader := ggio.NewDelimitedReader(s, network.MessageSizeMax)
	for {
		msg := &proto.Message{}
		if err := reader.ReadMsg(msg); err != nil {
			if err != io.EOF {
				if err := s.Reset(); err != nil {
					p2p.logger.WithField("error", err).Error("Reset stream")
				}
			}
			return
		}

		p2p.recvQ <- &net.MessageStream{
			Message: msg,
			Stream:  s,
		}
	}
}

// waitMsg wait the incoming messages within time duration.
func waitMsg(stream io.Reader, timeout time.Duration) *proto.Message {
	reader := ggio.NewDelimitedReader(stream, network.MessageSizeMax)
	rs := make(chan *proto.Message)
	go func() {
		msg := &proto.Message{}
		if err := reader.ReadMsg(msg); err == nil {
			rs <- msg
		} else {
			rs <- nil
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	select {
	case r := <-rs:
		cancel()
		return r
	case <-ctx.Done():
		cancel()
		return nil
	}
}

func (p2p *P2P) send(s network.Stream, message *proto.Message) error {
	deadline := time.Now().Add(sendTimeout)

	if err := s.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	writer := ggio.NewDelimitedWriter(s)
	if err := writer.WriteMsg(message); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}

	return nil
}
