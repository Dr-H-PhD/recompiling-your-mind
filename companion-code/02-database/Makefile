.PHONY: run test db-up db-down

# Start database
db-up:
	docker-compose up -d

# Stop database
db-down:
	docker-compose down

# Run the application
run: db-up
	@sleep 2
	go run main.go

# Run tests
test: db-up
	@sleep 2
	go test -v ./...

# Clean up
clean:
	docker-compose down -v
