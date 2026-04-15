BINARY_NAME=bot.exe
MAIN_PATH=./cmd/bot/main.go

all: build run

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

run:
	./$(BINARY_NAME)

clean:
	del $(BINARY_NAME)

build-linux:
	set GOOS=linux&& set GOARCH=amd64&& go build -o bot_linux $(MAIN_PATH)