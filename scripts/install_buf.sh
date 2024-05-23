#!/bin/bash

# Define the version of Buf to install
VERSION="v1.32.0"

# Install Buf for managing protocol buffers.
echo "Installing Buf version $VERSION"
GO111MODULE=on GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@$VERSION

echo "Buf installed successfully."