TAGS = webkit2_41

.PHONY: dev build test vet

dev:
	wails dev -tags $(TAGS)

build:
	wails build -tags $(TAGS)

test:
	go test ./internal/...

vet:
	go vet -tags $(TAGS) ./...