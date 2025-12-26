.PHONY: build test clean proto tidy run

# Build the application
build:
	go build ./...

# Run tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Generate protocol buffers
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		contracts/order/order.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		contracts/admin/admin.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		contracts/events/produced/uois_events.proto

# Clean generated files
clean:
	rm -f contracts/order/*.pb.go
	rm -f contracts/admin/*.pb.go
	rm -f contracts/events/produced/*.pb.go
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Run the application
run:
	go run cmd/server/main.go

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Verify build and tests
verify: build test
	@echo "✅ Build and tests passed"

