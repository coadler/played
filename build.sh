#!/bin/bash

set -euo pipefail
cd "$(dirname "$0")"

VERSION="$(git describe --dirty --always)"
if [[ $VERSION == *-dirty ]]; then
  # We need to ensure the image is loaded again so we give the image a unique
  # name from other images based on this dirty commit.
  VERSION+="-$(head -c 5 < /dev/urandom | base32)"
fi

tgt="rg.fr-par.scw.cloud/tatsu/played:$VERSION"
docker build -t "$tgt" .
docker push "$tgt"

echo "New image URI: $tgt"
