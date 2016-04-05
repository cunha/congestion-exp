BIN= $(shell basename $$PWD)
GP= $(shell dirname $(shell dirname $$PWD))

all:	export GOPATH=$(GP)
all:	*.go
	go build $(BIN)

xcompile: export GOOS=linux
xcompile: export GOARCH=arm
xcompile: all

clean:
	rm $(BIN)
