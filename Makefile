APP_NAME=droneops-sim
BUILD_DIR=build
BIN=$(BUILD_DIR)/$(APP_NAME)

.PHONY: all build run clean docker test

all: build

build:
	go build -o $(BIN) ./cmd/$(APP_NAME)

run:
	$(MAKE) build
	$(BIN) simulate --config config/simulation.yaml --schema schemas/simulation.cue --print-only

docker:
	docker build -t $(APP_NAME):latest .

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...