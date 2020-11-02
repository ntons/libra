SUBDIRS = certs

.PHONY: all $(SUBDIRS)

all: $(SUBDIRS)
	docker-compose build

$(SUBDIRS):
	@$(MAKE) -C $@

