VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"
BINARY  := brewbot

.PHONY: build run release clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

# Tag and push a release: make release TAG=v1.0.0
release:
	@test -n "$(TAG)" || (echo "usage: make release TAG=v1.0.0" && exit 1)
	git tag $(TAG)
	git push origin $(TAG)
	@echo "Released $(TAG)"

clean:
	rm -f $(BINARY)
