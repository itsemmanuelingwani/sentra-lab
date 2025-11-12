package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	target string
}

func NewClient(target string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
	}

	return &Client{
		conn:   conn,
		target: target,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetConnection() *grpc.ClientConn {
	return c.conn
}

type ClientOption func(*clientOptions)

type clientOptions struct {
	timeout    time.Duration
	maxRetries int
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(opts *clientOptions) {
		opts.timeout = timeout
	}
}

func WithMaxRetries(maxRetries int) ClientOption {
	return func(opts *clientOptions) {
		opts.maxRetries = maxRetries
	}
}