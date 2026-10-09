package grpcserver

import (
	"os"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"grpcserver",
	fx.Provide(
		NewPenneGRPCServer,
	),
	fx.Invoke(
		StartGRPCServerFromEnv,
	),
)

func StartGRPCServerFromEnv(srv *PenneGRPCServer, lc fx.Lifecycle, log *zap.Logger) error {
	addr := os.Getenv("GRPC_ADDR")
	if addr == "" {
		addr = ":50051"
	}
	_, err := StartGRPCServer(srv, addr, lc)
	return err
}
