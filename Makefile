.PHONY: clean run deploy build.local build.linux

BINARY        ?= feed
SOURCES       = $(shell find . -name '*.go')
STATICS       = $(shell find . -name '*.tmpl')
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -w -s

default: run

clean:
	rm -rf build

run: build.local
	SMTP_PORT=587 \
	SMTP_FROM=news@glv.one \
	./build/$(BINARY)

test:
	go test

deploy: build.linux
	rsync build/linux/$(BINARY) ec2-user@glv:feed/$(BINARY)-next
	ssh ec2-user@glv 'cp feed/$(BINARY) feed/$(BINARY)-old'
	ssh ec2-user@glv 'mv feed/$(BINARY)-next feed/$(BINARY)'
	ssh ec2-user@glv 'sudo systemctl restart $(BINARY)'

rollback:
	ssh ec2-user@glv 'mv feed/$(BINARY)-old feed/$(BINARY)'
	ssh ec2-user@glv 'sudo systemctl restart $(BINARY)'

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)

build/$(BINARY): $(SOURCES) $(STATICS)
	CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): $(SOURCES) $(STATICS)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .
