.PHONY: run test tidy vet lint verify verify-integration devbox-bootstrap devbox-up devbox-down devbox-suspend devbox-pause devbox-resume devbox-destroy devbox-status build-bundle run-bundle

MODULE_DIRS := $(shell go list -f '{{.Dir}}' -m all | grep -v '/pkg/mod/')
DEVBOX_DIR := $(if $(wildcard ./devops/devbox/Makefile),./devops/devbox,../devops/devbox)
BUNDLE ?= service

run:
	@echo "Run service (all compiled-in features)"
	@cd cmd/service && go run .

run-bundle:
	@echo "Run bundle $(BUNDLE)"
	@cd cmd/$(BUNDLE) && go run .

build-bundle:
	@echo "Build bundle $(BUNDLE)"
	@mkdir -p bin
	@cd cmd/$(BUNDLE) && go build -o ../../bin/$(BUNDLE) .

test:
	@echo "Run tests"
	@for dir in $(MODULE_DIRS); do \
		echo "test $$dir..."; \
		(cd $$dir && go test ./...); \
	done

tidy:
	@echo "Run go mod tidy"
	@for dir in $(MODULE_DIRS); do \
		echo "tidy $$dir..."; \
		(cd $$dir && go mod tidy); \
	done

vet:
	@echo "Run go vet"
	@for dir in $(MODULE_DIRS); do \
		echo "vet $$dir..."; \
		(cd $$dir && go vet ./...); \
	done

lint:
	@echo "Run lint (autofix)"
	@for dir in $(MODULE_DIRS); do \
		echo "lint $$dir..."; \
		(cd $$dir && golangci-lint run --fix ./...); \
	done

verify:
	@echo "Run backend verify gate"
	@bash verify.sh

verify-integration: verify

devbox-bootstrap:
	@$(MAKE) -C $(DEVBOX_DIR) bootstrap

devbox-up:
	@$(MAKE) -C $(DEVBOX_DIR) up

devbox-down:
	@$(MAKE) -C $(DEVBOX_DIR) down

devbox-suspend:
	@$(MAKE) -C $(DEVBOX_DIR) suspend

devbox-pause: devbox-suspend

devbox-resume:
	@$(MAKE) -C $(DEVBOX_DIR) resume

devbox-destroy:
	@$(MAKE) -C $(DEVBOX_DIR) destroy

devbox-status:
	@$(MAKE) -C $(DEVBOX_DIR) status
