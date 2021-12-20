.PHONY: all
all: $(mage_target)

include .mage/gen.mk
include $(mage_targets)
