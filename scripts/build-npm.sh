#!/bin/sh
# Cross-compiles and writes one npm package per platform to dist/npm.
set -eu
version=$(node -p 'require("./package.json").version')
rm -rf dist/npm
for target in darwin/arm64/darwin/arm64 darwin/amd64/darwin/x64 linux/arm64/linux/arm64 linux/amd64/linux/x64 windows/amd64/win32/x64; do
  IFS=/ read -r goos goarch os cpu <<TARGET
$target
TARGET
  dir="dist/npm/samecontact-$os-$cpu"
  exe=samecontact
  [ "$goos" = windows ] && exe=samecontact.exe
  mkdir -p "$dir"
  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -ldflags='-s -w -buildid=' -o "$dir/$exe" .
  cp LICENSE "$dir/LICENSE"
  cat > "$dir/package.json" <<JSON
{
  "name": "samecontact-$os-$cpu",
  "version": "$version",
  "description": "The samecontact binary for $os-$cpu.",
  "license": "MIT",
  "repository": { "type": "git", "url": "git+https://github.com/GillMacedo/samecontact.git" },
  "os": ["$os"],
  "cpu": ["$cpu"],
  "preferUnplugged": true
}
JSON
done
