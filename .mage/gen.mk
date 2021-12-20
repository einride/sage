mage_folder := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
mage_targets_path := $(mage_folder)/gen
mage_targets := $(mage_targets_path)/targets.mk
mage := $(mage_targets_path)/local_mage

$(mage_targets): $(mage_folder)/go.mod $(shell find $(mage_folder)/.. -type f -name '*.go')
	@git clean $(mage_targets_path)/ -fdx
	@mkdir -p $(mage_targets_path)
	@cd $(mage_folder) && \
		go mod tidy && \
		go run main.go -compile $(mage) && \
		$(mage) generateMakefile $(@)

.PHONY: mage-clean
mage-clean:
	@git clean -fdx $(mage_targets) ./.tools
