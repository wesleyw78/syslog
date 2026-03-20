package ingest

import (
	"context"
	"errors"
	"net"
	"time"
)

const defaultUDPBufferSize = 64 * 1024

var errUDPListenerNotStarted = errors.New("udp listener not started")

type Handler func(ctx context.Context, payload []byte, addr net.Addr) error

type UDPListener struct {
	addr       string
	handler    Handler
	conn       net.PacketConn
	bufferSize int
}

func NewUDPListener(addr string, handler Handler) *UDPListener {
	if addr == "" {
		addr = ":1514"
	}

	return &UDPListener{
		addr:       addr,
		handler:    handler,
		bufferSize: defaultUDPBufferSize,
	}
}

func (l *UDPListener) Start() error {
	if l.conn != nil {
		return nil
	}

	conn, err := net.ListenPacket("udp", l.addr)
	if err != nil {
		return err
	}

	l.conn = conn
	return nil
}

func (l *UDPListener) ReadOnce(ctx context.Context) ([]byte, net.Addr, error) {
	if l.conn == nil {
		return nil, nil, errUDPListenerNotStarted
	}

	buffer := make([]byte, l.bufferSize)
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		if err := l.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return nil, nil, err
		}

		size, addr, err := l.conn.ReadFrom(buffer)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			return nil, nil, err
		}

		payload := append([]byte(nil), buffer[:size]...)
		return payload, addr, nil
	}
}

func (l *UDPListener) Serve(ctx context.Context) error {
	if err := l.Start(); err != nil {
		return err
	}

	for {
		payload, addr, err := l.ReadOnce(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if errors.Is(err, net.ErrClosed) && ctx.Err() != nil {
				return nil
			}
			return err
		}

		if l.handler == nil {
			continue
		}

		if err := l.handler(ctx, payload, addr); err != nil {
			return err
		}
	}
}

func (l *UDPListener) Close() error {
	if l.conn == nil {
		return nil
	}

	err := l.conn.Close()
	l.conn = nil
	return err
}

func (l *UDPListener) Addr() net.Addr {
	if l.conn == nil {
		return nil
	}

	return l.conn.LocalAddr()
}
