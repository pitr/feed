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
	SMTP_USERNAME=$U \
	SMTP_PASSWORD=$P \
	SMTP_HOST=email-smtp.eu-west-1.amazonaws.com \
	SMTP_FROM=news@glv.one \
	./build/$(BINARY)

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
	CGO_ENABLED=1 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .
