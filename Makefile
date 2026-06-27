# Orbit Sources — official WASM plugin sources.

ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
DIST := $(ROOT)dist
INSTALL_PLUGINS_DIR ?= $(ROOT)../plugins
comma := ,

PLUGINS := $(sort $(notdir $(patsubst %/,%,$(dir $(wildcard plugins/*/*/Makefile)))))
plugin_dir = $(patsubst %/,%,$(dir $(firstword $(wildcard plugins/*/$(1)/Makefile))))

PLUGIN ?=
ifneq ($(strip $(PLUGIN)),)
  SELECTED := $(strip $(subst $(comma), ,$(PLUGIN)))
else
  SELECTED := $(PLUGINS)
endif

ORBIT_PACK := cd cmd/orbit-pack && go run .

.DEFAULT_GOAL := help

.PHONY: help list build build-all package package-all orbit orbit-all sync sync-all clean clean-all test-native test-native-all

help:
	@echo "Orbit WASM plugin sources"
	@echo ""
	@echo "Discovered plugins: $(PLUGINS)"
	@echo "Build outputs: $(DIST)/<id>/"
	@echo ""
	@echo "Common commands:"
	@echo "  make list"
	@echo "  make build PLUGIN=<id>"
	@echo "  make package PLUGIN=<id>"
	@echo "  make orbit PLUGIN=<id>"
	@echo "  make try PLUGIN=<id> ROUTE=<route> PARAMS='{}'"
	@echo "  make try-wasm PLUGIN=<id> ROUTE=<route> PARAMS='{}'"
	@echo "  make dev PLUGIN=<id> ROUTE=<route> PARAMS='{}'"
	@echo ""
	@echo "All plugins:"
	@echo "  make build-all | package-all | orbit-all | clean-all | test-native-all"

list:
	@printf '%s\n' $(PLUGINS)

define assert_plugins
$(if $(SELECTED),,$(error No plugins found. Add plugins/<category>/<id>/Makefile))
$(foreach p,$(SELECTED),$(if $(filter $(p),$(PLUGINS)),,$(error Unknown plugin "$(p)". Available: $(PLUGINS))))
endef

define PLUGIN_RULES
.PHONY: build-$(1) package-$(1) orbit-$(1) sync-$(1) clean-$(1) test-native-$(1) try-$(1) try-wasm-$(1) dev-$(1)

build-$(1):
	@echo "-> $(1): build"
	@$(MAKE) -C $(call plugin_dir,$(1)) build

package-$(1): build-$(1)
	@echo "-> $(1): package"
	@$(MAKE) -C $(call plugin_dir,$(1)) package

orbit-$(1): package-$(1)
	@echo "-> $(1): extension.orbit"
	@$(ORBIT_PACK) -dist "$(DIST)/$(1)" -src "$(ROOT)$(call plugin_dir,$(1))"

sync-$(1): package-$(1)
	@echo "-> installing $(1) to $(INSTALL_PLUGINS_DIR)/$(1)"
	@rm -rf "$(INSTALL_PLUGINS_DIR)/$(1)"
	@mkdir -p "$(INSTALL_PLUGINS_DIR)/$(1)"
	@cp -R "$(DIST)/$(1)/"* "$(INSTALL_PLUGINS_DIR)/$(1)/"

clean-$(1):
	@echo "-> $(1): clean"
	@$(MAKE) -C $(call plugin_dir,$(1)) clean

test-native-$(1):
	@echo "-> $(1): test-native"
	@$(MAKE) -C $(call plugin_dir,$(1)) test-native

try-$(1):
	@bash scripts/try.sh $(1) native

try-wasm-$(1):
	@bash scripts/try.sh $(1) wasm

dev-$(1):
	@bash scripts/try.sh $(1) runtime
endef

$(foreach p,$(PLUGINS),$(eval $(call PLUGIN_RULES,$(p))))

ifneq ($(strip $(PLUGIN)),)
  BUILD_TARGETS := $(foreach p,$(SELECTED),build-$(p))
  PACKAGE_TARGETS := $(foreach p,$(SELECTED),package-$(p))
  ORBIT_TARGETS := $(foreach p,$(SELECTED),orbit-$(p))
  SYNC_TARGETS := $(foreach p,$(SELECTED),sync-$(p))
  CLEAN_TARGETS := $(foreach p,$(SELECTED),clean-$(p))
  TEST_TARGETS := $(foreach p,$(SELECTED),test-native-$(p))
else
  BUILD_TARGETS := $(foreach p,$(PLUGINS),build-$(p))
  PACKAGE_TARGETS := $(foreach p,$(PLUGINS),package-$(p))
  ORBIT_TARGETS := $(foreach p,$(PLUGINS),orbit-$(p))
  SYNC_TARGETS := $(foreach p,$(PLUGINS),sync-$(p))
  CLEAN_TARGETS := $(foreach p,$(PLUGINS),clean-$(p))
  TEST_TARGETS := $(foreach p,$(PLUGINS),test-native-$(p))
endif

build: $(call assert_plugins) build-all
build-all: $(BUILD_TARGETS)

package: $(call assert_plugins) package-all
package-all: $(PACKAGE_TARGETS)

orbit: $(call assert_plugins) orbit-all
orbit-all: $(ORBIT_TARGETS)

sync: $(call assert_plugins) sync-all
sync-all: $(SYNC_TARGETS)
	@echo "installed -> $(INSTALL_PLUGINS_DIR)"

clean: $(call assert_plugins) clean-all
clean-all: $(CLEAN_TARGETS)

test-native: $(call assert_plugins) test-native-all
test-native-all: $(TEST_TARGETS)

try: $(call assert_plugins) $(foreach p,$(SELECTED),try-$(p))
try-wasm: $(call assert_plugins) $(foreach p,$(SELECTED),try-wasm-$(p))
dev: $(call assert_plugins) $(foreach p,$(SELECTED),dev-$(p))
