mage_folder := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
mage_generated_path := $(mage_folder)/gen
mage_targets_file := $(mage_generated_path)/targets.mk
mage := $(mage_generated_path)/local-mage

include $(mage_targets_file)

$(mage_targets_file): $(mage_folder)/go.mod $(mage_folder)/*.go
	@git clean -fdx $(mage_generated_path)
	@mkdir -p $(mage_generated_path)
	@cd $(mage_folder) && \
		go mod tidy && \
		go run main.go -compile $(mage) && \
		$(mage) generateMakefile $(@)

.PHONY: mage-clean
mage-clean:
	@git clean -fdx $(mage_generated_path) ./.tools
