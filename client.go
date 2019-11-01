package played

import (
	"context"

	"golang.org/x/xerrors"
	"nhooyr.io/websocket"
)

type Client struct {
	addr  string
	conn  *websocket.Conn
	fails int
}

func NewClient(ctx context.Context, addr string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, addr, nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to dial played server: %w", err)
	}

	return &Client{
		addr: addr,
		conn: conn,
	}, nil
}

func (c *Client) WritePresence(ctx context.Context, buf []byte) error {
	err := c.conn.Write(ctx, websocket.MessageBinary, buf)
	if err != nil {
		if c.fails > 5 {
			return xerrors.Errorf("failed to connect to played: %w", err)
		}
		c.fails++

		conn, _, err := websocket.Dial(ctx, c.addr, nil)
		if err != nil {
			return xerrors.Errorf("failed to reinitialize played conn: %w", err)
		}

		c.conn = conn
		return c.WritePresence(ctx, buf)
	}

	return nil
}
