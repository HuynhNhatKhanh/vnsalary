.PHONY: test race bench fuzz lint golden build fmt vet check clean

test:
	go test -v -cover ./...

# The race detector needs a 64-bit platform (e.g. linux/amd64); CI runs it.
race:
	go test -v -race -cover ./...

bench:
	go test -bench=. -benchmem -run=^$$ .

fuzz:
	go test -run=^$$ -fuzz=^FuzzGrossToNet_NetToGross$$ -fuzztime=60s .

lint:
	golangci-lint run ./...

# Rewrite the CLI output snapshots after an intended change; review the diff.
golden:
	go test ./cmd/vnsalary -run TestRun_Golden -update

build:
	go build -o bin/vnsalary ./cmd/vnsalary

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin coverage.out

check: fmt vet lint test
