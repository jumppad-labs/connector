#!/bin/sh

# Install UPX
wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz
tar -xf upx-3.96-amd64_linux.tar.xz

# Compress files
./upx $1

rm -rf upx-3.96-amd64_linux
rm -rf ./upx
rm -rf upx-3.96-amd64_linux.tar.xz
