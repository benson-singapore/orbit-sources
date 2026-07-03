PLUGIN_ID = jimeng
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
	echo '{"action":"fetch","data":{"channelId":"trending","route":"/jimeng/explore","params":$(PARAMS)}}' | go run .

CHANNEL ?= trending
ROUTE ?= /jimeng/explore
PARAMS ?= {"page":"1","workType":"video,image,canvas"}
