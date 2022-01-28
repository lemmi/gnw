#!/bin/bash

set -e

VERSION=${VERSION:-$(git describe --tags)}
NAME=$(go list -f '{{.Target}}' | xargs basename)

target() {
	: ${GOOS:=$(go env GOOS)}
	: ${GOARCH:=$(go env GOARCH)}

	echo "${NAME}-${GOOS}-${GOARCH}${GOARM:+v$GOARM}-$VERSION"
}

build() {
	t=$(target)
	echo building "$t"
	mkdir -p "dist/$t"
	go build -tags netgo -o "dist/$t"
	tar czf "dist/$t.tar.gz" -C dist "$t"
}

rm -rf dist
build
GOARCH=arm64 build
GOARCH=arm GOARM=6 build
GOARCH=arm GOARM=7 build
gh release create ${VERSION} dist/*tar.gz
rm -rf dist
