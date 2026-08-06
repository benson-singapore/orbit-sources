PLUGIN_ID = hongguoguo
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

clean:
	rm -rf $(DIST_DIR)

test-native:
	echo '{"action":"fetch","data":{"channelId":"$(CHANNEL)","route":"$(ROUTE)","params":$(PARAMS),"vars":{"baseURL":"$(BASE_URL)"}}}' | go run .

CHANNEL ?= jingxuan
ROUTE ?= /hongguoguo/list/:type
PARAMS ?= {"type":"jingxuanduanju","page":"1"}
BASE_URL ?= https://www.hongguoguo.tv
