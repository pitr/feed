.PHONY: clean run build.local build.linux

BINARY        ?= feed
SOURCES       = $(shell find . -name '*.go')
STATICS       = $(shell find . -name '*.tmpl')
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -w -s

default: run

clean:
	rm -rf build

run: build.local
	./build/$(BINARY)

test:
	go test

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)

build/$(BINARY): $(SOURCES) $(STATICS)
	CGO_ENABLED=0 go1.16beta1 build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): $(SOURCES) $(STATICS)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go1.16beta1 build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .
