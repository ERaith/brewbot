VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"
BINARY  := brewbot

.PHONY: build run deploy release clean

## Local build
build:
	go build $(LDFLAGS) -o $(BINARY) .

run: build
	./$(BINARY)

## Deploy: pull main, rebuild container, restart
deploy:
	git pull origin main
	VERSION=$(VERSION) docker compose up --build -d
	docker compose logs -f --tail=50

## Cut a release: make release TAG=v1.0.0
release:
	@test -n "$(TAG)" || (echo "usage: make release TAG=v1.0.0" && exit 1)
	git tag $(TAG)
	git push origin $(TAG)
	@echo "Released $(TAG) — run 'make deploy' on the server"

clean:
	rm -f $(BINARY)
	docker compose down
