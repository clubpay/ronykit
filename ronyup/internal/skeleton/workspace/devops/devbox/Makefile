.PHONY: bootstrap up down destroy status kubeconfig services ssh help

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
export KUBECONFIG ?= $(shell bash "$(ROOT)/scripts/resolve-kubeconfig.sh")

help:
	@echo "Devbox targets:"
	@echo "  make bootstrap  - verify host tools (kubectl, helm, helmfile, yq; + vagrant when cluster.mode=vagrant)"
	@echo "  make up         - connect to cluster and install enabled services"
	@echo "  make down       - halt VM (vagrant) or remove devbox services (existing cluster)"
	@echo "  make destroy    - destroy VM (vagrant) or remove devbox services (existing cluster)"
	@echo "  make kubeconfig - refresh shared/kubeconfig (vagrant mode only)"
	@echo "  make services   - apply Helmfile releases from config.yaml"
	@echo "  make status     - cluster / vagrant status"
	@echo ""
	@echo "Edit config.yaml: cluster.mode (existing|vagrant) and service toggles."

bootstrap:
	@bash scripts/bootstrap.sh

up: bootstrap
	@bash scripts/up.sh

down:
	@bash scripts/down.sh

destroy:
	@bash scripts/destroy.sh

kubeconfig:
	@bash scripts/kubeconfig.sh

services:
	@bash scripts/install-services.sh

status:
	@if [ "$$(yq -r '.cluster.mode // "existing"' config.yaml)" = "vagrant" ]; then vagrant status; fi
	@kubectl get nodes 2>/dev/null || true

ssh:
	@vagrant ssh
