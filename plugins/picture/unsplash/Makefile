PLUGIN_ID = unsplash
DIST_DIR  = ../../../dist/$(PLUGIN_ID)
OUTPUT    = $(DIST_DIR)/plugin.wasm

.PHONY: build package clean test-native

build:
	@mkdir -p $(DIST_DIR)
	GOOS=wasip1 GOARCH=wasm go build -o $(OUTPUT) .
	@echo "built $(OUTPUT)"

package: build
	cp manifest.json $(DIST_DIR)/
	cp README.md $(DIST_DIR)/
	@mkdir -p $(DIST_DIR)/assets

clean:
	rm -rf $(DIST_DIR)

test-native:
	echo '{"action":"fetch","data":{"channelId":"$(CHANNEL)","route":"$(ROUTE)","params":$(PARAMS),"vars":{"cookie":"$(COOKIE)"}}}' | go run .

CHANNEL ?= nature
ROUTE ?= /unsplash/topics
PARAMS ?= {"topic":"nature","page":"1","size":"20"}
COOKIE ?=
