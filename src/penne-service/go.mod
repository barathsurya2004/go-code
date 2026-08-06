module github.com/barathsurya2004/go-code/penne-service

go 1.25.5

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/barathsurya2004/go-code/pkg v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.3
	github.com/mark3labs/mcp-go v0.57.0
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/barathsurya2004/go-code/pkg => ../../pkg
