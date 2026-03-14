package grader

import (
	"context"

	"github.com/jjudge-oj/grader/api/graderpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps a gRPC connection to the grader service.
type Client struct {
	conn   *grpc.ClientConn
	client graderpb.GraderClient
}

// NewClient dials the grader gRPC service at the given address.
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		client: graderpb.NewGraderClient(conn),
	}, nil
}

// Grade calls the grader service to compare output with expected output.
// Returns true if the output is correct.
func (c *Client) Grade(ctx context.Context, output, expectedOutput, graderType string) (bool, error) {
	resp, err := c.client.Grade(ctx, &graderpb.GraderRequest{
		Output:         output,
		ExpectedOutput: expectedOutput,
		UseGrader:      graderType,
	})
	if err != nil {
		return false, err
	}
	return resp.GetOk(), nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
