#!/bin/bash

set -euo pipefail
cd "$(dirname "$0")"

docker build -t rg.fr-par.scw.cloud/tatsu/played .
docker push rg.fr-par.scw.cloud/tatsu/played
