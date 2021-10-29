.PHONY: all
all: $(mage_version_lock)

mage_folder := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/.mage)
mage_gen := $(mage_folder)/gen
mage_tools = $(shell cd $(mage_folder) && go list -m -f '{{ .Version }}' github.com/einride/mage-tools)
magefile = $(shell md5sum $(mage_folder)/magefile.go)
mage_version_lock = $(mage_gen)/$(mage_tools)-$(magefile)
mage_target := $(mage_gen)/targets.mk
mage := go run mage.go

include $(mage_target)

# The lockfile is a combination of the mage-tool version and magefile md5sum
# to support regeneration when mage-tools are bumped or magefile is changed.
$(mage_version_lock):
	$(info [$@] generating version lock file)
	@git clean $(mage_folder)/ -fdx
	@mkdir -p $(mage_gen)
	@touch $(mage_version_lock)

$(mage_target): $(mage_version_lock)
	$(info [$@] generating targets...)
	@cd $(mage_folder) && \
		go mod tidy && \
		$(mage) generateMakefile $(@)
