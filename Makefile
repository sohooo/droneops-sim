APP_NAME=droneops-sim
BUILD_DIR=build

.PHONY: all build run clean docker test

all: build

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run:
	$(BUILD_DIR)/$(APP_NAME) --config config/fleet.yaml --schema schemas/fleet.cue --print-only

docker:
	docker build -t $(APP_NAME):latest .

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...