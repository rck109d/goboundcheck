ifeq ($(OS),Windows_NT)
	BINARY_FILE=goboundcheck.exe
else
	BINARY_FILE=goboundcheck
endif

all: golangci gosec compile

golangci:
	golangci-lint run ./...

gosec:
	gosec -exclude-dir=testdata ./...

compile:
	go build -o ${BINARY_FILE} ./cmd/goboundcheck

test:
	go test ./...

clean:
	go clean
	rm ${BINARY_FILE}
