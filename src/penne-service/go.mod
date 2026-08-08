module github.com/barathsurya2004/go-code/penne-service

go 1.25.5

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/barathsurya2004/go-code/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
)

require (
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace github.com/barathsurya2004/go-code/pkg => ../../pkg
