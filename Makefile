generate:
	@go generate ./...

build: generate
	@echo "====> Build bitfinex"
	@sh -c ./build.sh
