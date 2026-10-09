package grpcclient

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// Default tuning parameters for maximizing throughput and minimizing head-of-line blocking.
const (
	DefaultPoolSize              = 4
	DefaultInitialWindowSize     = 1024 * 1024     // 1MB stream flow control window
	DefaultInitialConnWindowSize = 4 * 1024 * 1024 // 4MB connection flow control window
	DefaultKeepAliveTime         = 15 * time.Second
	DefaultKeepAliveTimeout      = 5 * time.Second
)

var (
	grpcNewClient = grpc.NewClient
	connClose     = func(conn *grpc.ClientConn) error { return conn.Close() }
)

// PoolConfig holds configuration options for the gRPC ClientPool.
type PoolConfig struct {
	Target                string
	Size                  int
	InitialWindowSize     int32
	InitialConnWindowSize int32
	KeepAliveTime         time.Duration
	KeepAliveTimeout      time.Duration
	DialOptions           []grpc.DialOption
}

// ClientPool manages a pool of gRPC client connections to a single target.
// Requests are load-balanced across the connections via round-robin to avoid
// HTTP/2 stream concurrency saturation and flow-control bottlenecks.
type ClientPool struct {
	conns []*grpc.ClientConn
	idx   uint64
}

// NewClientPool initializes a pool of gRPC connections to target with high-performance defaults.
func NewClientPool(cfg PoolConfig) (*ClientPool, error) {
	if cfg.Target == "" {
		return nil, fmt.Errorf("target address cannot be empty")
	}

	size := cfg.Size
	if size <= 0 {
		size = DefaultPoolSize
	}

	winSize := cfg.InitialWindowSize
	if winSize <= 0 {
		winSize = DefaultInitialWindowSize
	}

	connWinSize := cfg.InitialConnWindowSize
	if connWinSize <= 0 {
		connWinSize = DefaultInitialConnWindowSize
	}

	kaTime := cfg.KeepAliveTime
	if kaTime <= 0 {
		kaTime = DefaultKeepAliveTime
	}

	kaTimeout := cfg.KeepAliveTimeout
	if kaTimeout <= 0 {
		kaTimeout = DefaultKeepAliveTimeout
	}

	target := cfg.Target
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithInitialWindowSize(winSize),
		grpc.WithInitialConnWindowSize(connWinSize),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                kaTime,
			Timeout:             kaTimeout,
			PermitWithoutStream: true,
		}),
	}

	// Handle Unix Domain Socket targets: unix:///path/to/sock or unix:/path
	if strings.HasPrefix(target, "unix://") || strings.HasPrefix(target, "unix:") {
		sockPath := strings.TrimPrefix(target, "unix://")
		sockPath = strings.TrimPrefix(sockPath, "unix:")
		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sockPath)
		}))
		target = "passthrough:///" + sockPath
	}

	if len(cfg.DialOptions) > 0 {
		dialOpts = append(dialOpts, cfg.DialOptions...)
	}

	conns := make([]*grpc.ClientConn, 0, size)
	for i := 0; i < size; i++ {
		conn, err := grpcNewClient(target, dialOpts...)
		if err != nil {
			// Clean up any established connections
			for _, c := range conns {
				_ = c.Close()
			}
			return nil, fmt.Errorf("failed to create gRPC client connection %d: %w", i, err)
		}
		conns = append(conns, conn)
	}

	return &ClientPool{
		conns: conns,
	}, nil
}

// Get returns the next connection in the pool using atomic round-robin selection.
func (p *ClientPool) Get() *grpc.ClientConn {
	if len(p.conns) == 0 {
		return nil
	}
	next := atomic.AddUint64(&p.idx, 1)
	return p.conns[(next-1)%uint64(len(p.conns))]
}

// Size returns the number of connections in the pool.
func (p *ClientPool) Size() int {
	return len(p.conns)
}

// Close closes all connections maintained by the pool.
func (p *ClientPool) Close() error {
	var firstErr error
	for _, conn := range p.conns {
		if err := connClose(conn); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// WithUserMetadata attaches user identity and request correlation metadata to the gRPC outgoing context.
func WithUserMetadata(ctx context.Context, userUUID, requestID string) context.Context {
	md := metadata.Pairs()
	if userUUID != "" {
		md.Set("x-user-uuid", userUUID)
	}
	if requestID != "" {
		md.Set("x-request-id", requestID)
	}
	return metadata.NewOutgoingContext(ctx, md)
}
