build-api:
	
	go build -o pt-api cmd/payload-tracker-api/main.go

build-consumer:

	go build -o pt-consumer cmd/payload-tracker-consumer/main.go

build-all:

	go build -o pt-api cmd/payload-tracker-api/main.go
	go build -o pt-consumer cmd/payload-tracker-consumer/main.go

lint:
	gofmt -l .
	gofmt -s -w .

test:
	go test -p 1 -v ./...

run-seed:
	go build -o pt-seeder tools/db-seeder/main.go
	./pt-seeder

run-migration:
	go build -o pt-migration internal/migration/main.go
	./pt-migration
