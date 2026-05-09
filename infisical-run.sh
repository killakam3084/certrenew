#!/bin/sh
# Entrypoint wrapper: injects secrets via Infisical CLI then runs certrenew.
# Variables are expanded at container runtime from env_file, not at compose parse time.
set -e

exec infisical run \
  --projectId="${INFISICAL_PROJECT_ID}" \
  --env="${INFISICAL_ENV:-prod}" \
  -- ./certrenew "$@"

