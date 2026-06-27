PLUGIN_ID = youtube
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
	echo '{"action":"fetch","data":{"channelId":"$(CHANNEL_ID)","playlistId":"$(PLAYLIST_ID)","route":"$(ROUTE)","params":$(PARAMS)}}' | go run .

CHANNEL_ID ?= UCDwDMPOZfxVV0x_dz0eQ8KQ
PLAYLIST_ID ?= PLqQ1RwlxOgeLTJ1f3fNMSwhjVgaWKo_9Z
ROUTE ?= /youtube/channel/:channelId
PARAMS ?= {}
