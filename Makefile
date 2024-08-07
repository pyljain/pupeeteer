gen-proto:
	protoc --go_out=. --go-grpc_out=. ./pkg/contract/agent.proto
	protoc --go_out=. --go-grpc_out=. ./pkg/contract/cli.proto

build:
	go build

run-pm: build
	./ptr start puppetmaster

run-puppet: build
	./ptr start puppet --name=pinnochio

