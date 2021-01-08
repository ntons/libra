.PHONY: all help apiproxy edgeproxy

# all directory names
TARGETS = $(patsubst %/,%,$(dir $(sort $(dir $(wildcard ./*/*)))))

all: $(TARGETS)

help:
	@echo "Usage: make [target] | kubectl [-n namespace] apply -f -"

$(TARGETS):
	@kubectl create cm $@ --dry-run=client -o yaml --from-file=./$@/
	@echo '---'

