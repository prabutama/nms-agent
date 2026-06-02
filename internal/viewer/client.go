package viewer

import (
	"bufio"
	"encoding/json"
	"net"
)

type Client struct {
	conn net.Conn
	dec  *json.Decoder
}

func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, dec: json.NewDecoder(bufio.NewReader(conn))}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) Next() (Message, error) {
	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return Message{}, err
	}
	return msg, nil
}
