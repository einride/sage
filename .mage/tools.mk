mage_dir := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
mage_tools_path := $(mage_dir)/tools
mage := $(mage_tools_path)/mgmake/magefile

$(mage): $(mage_dir)/go.mod $(shell find $(mage_dir)/.. -type f -name '*.go')
	@cd $(mage_dir) && go run ../main.go gen

.PHONY: mage-clean
mage-clean:
	@git clean -fdx $(mage_dir)
