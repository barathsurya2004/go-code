module github.com/barathsurya2004/go-code/gateway

go 1.26.0

replace github.com/barathsurya2004/go-code/pkg => ../../pkg

require (
	github.com/barathsurya2004/go-code/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
	google.golang.org/grpc v1.84.0
)

require (
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261012220534-4341edf3f941 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
