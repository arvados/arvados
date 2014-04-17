KEEP_BIN = ./bin/keep

SRC = keep.go
TEST_SRC = keep_test.go

all:	$(KEEP_BIN)

$(KEEP_BIN):
	go build
	mkdir -p ./bin
	mv keep $(KEEP_BIN)

test:	$(SRC) $(TEST_SRC)
	go test
