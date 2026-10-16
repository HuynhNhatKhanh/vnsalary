#!/usr/bin/env bash
# Builds the vnsalary CLI for each release target into dist/, with checksums.
# Usage: scripts/build-release.sh vX.Y.Z   (used by .github/workflows/release.yml)
set -euo pipefail

version="${1:?usage: $0 vX.Y.Z}"
targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

rm -rf dist && mkdir -p dist
for target in $targets; do
  os="${target%/*}"
  arch="${target#*/}"
  name="vnsalary_${version}_${os}_${arch}"
  ext=""
  if [ "$os" = windows ]; then ext=".exe"; fi

  mkdir -p "dist/$name"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
    -ldflags "-s -w -X main.version=${version}" \
    -o "dist/$name/vnsalary$ext" ./cmd/vnsalary
  cp LICENSE README.md "dist/$name/"

  if [ "$os" = windows ]; then
    (cd dist && zip -qr "$name.zip" "$name")
  else
    tar -C dist -czf "dist/$name.tar.gz" "$name"
  fi
  rm -rf "dist/${name:?}"
done

(cd dist && sha256sum ./*.tar.gz ./*.zip | sed 's| \./| |' > checksums.txt)
ls -l dist
