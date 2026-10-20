module github.com/barathsurya2004/go-code/pulse-service

go 1.26.0

replace github.com/barathsurya2004/go-code/pkg => ../../pkg

require (
	github.com/barathsurya2004/go-code/pkg v0.0.0-00010101000000-000000000000
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.12.3
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
)

require (
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)
