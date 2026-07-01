# Helm chart values for devbox services.

Releases are installed by `../scripts/helmfile-apply.sh` using plain `helm upgrade --install` (no Helmfile or helm-diff plugin).

Toggle services in `../config.yaml`, then run `make services`.
