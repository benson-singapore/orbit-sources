PLUGIN_ID = rycjapi
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
	echo '{"action":"fetch","data":{"channelId":"$(CHANNEL)","route":"$(ROUTE)","params":$(PARAMS),"vars":{"apiURL":"$(API_URL)","siteName":"$(SITE_NAME)"}}}' | go run .

CHANNEL ?= series-cn
ROUTE ?= /rycjapi/chapters/:id
PARAMS ?= {"id":"79534"}
API_URL ?= https://cj.rycjapi.com/api.php/provide/vod
SITE_NAME ?= TV-如意资源
