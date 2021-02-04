GOOS = linux
GOARCH = amd64

prom-cleaner:
	@echo "build prom-cleaner"
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build  -o build/prom-cleaner

clean:
	@echo "clean prom-cleaner"
	go clean -i && rm -rf build