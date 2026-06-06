build:
	go build -o project-stormlight ./cmd/project-stormlight

test:
	go test -v ./...
templ:
	templ generate ./...

run: 
	templ 
	go run ./cmd/project-stormlight