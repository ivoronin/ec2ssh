.PHONY: lint test build demo clean

lint:
	golangci-lint run

test:
	go test ./...

build:
	go build ./cmd/ec2ssh

demo:
	cd demo && vhs demo.vhs

clean:
	rm -rf dist/
