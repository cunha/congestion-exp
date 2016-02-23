BIN= congestion-exp
SRC= main.go cross_traffic.go utils.go

all:    $(SRC)
	go build $(BIN)
clean:
	rm $(BIN)
