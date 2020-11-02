TARGETS = agent render_template

.PHONY: all $(TARGETS)

all: $(TARGETS)

$(TARGETS):
	go build -o $@ github.com/ntons/libra/tools/$@

