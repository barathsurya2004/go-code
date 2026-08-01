module github.com/barathsurya2004/go-code/penne-service

go 1.25.0

require (
	github.com/barathsurya2004/go-code/pkg v0.0.0
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.12.3
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
)

require (
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
)

replace github.com/barathsurya2004/go-code/pkg => ../../pkg
