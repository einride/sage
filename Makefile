.PHONY: all
all: $(mage_target)

mage_folder := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/.mage)
mage_gen := $(mage_folder)/gen
mage_target := $(mage_gen)/targets.mk
mage := $(mage_gen)/local_mage

include $(mage_target)

$(mage_target): $(mage_folder)/go.mod $(mage_folder)/magefile.go
	$(info [$@] generating targets...)
	@git clean $(mage_gen)/ -fdx
	@mkdir -p $(mage_gen)
	@cd $(mage_folder) && \
		go mod tidy && \
		go run main.go -compile $(mage) && \
		$(mage) generateMakefile $(@)

clean:
	@git clean $(mage_gen)/ -fdx
