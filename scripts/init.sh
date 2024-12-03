#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
PROJECT_DIR="$(dirname "${SCRIPT_DIR}")"

cd ${PROJECT_DIR}

# gin-swagger
# generate swagger api automatically

# install swag
go install github.com/swaggo/swag/cmd/swag@latest
# init
swag init
