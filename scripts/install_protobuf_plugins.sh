#!/bin/bash

# Install Protobuf Plugins.

# Define the versions for the Protobuf plugins.
GRPC_GATEWAY_VERSION="v2.20.0"
OPENAPIV2_VERSION="v2.20.0"
PROTOC_GEN_GO_VERSION="v1.34.1"
PROTOC_GEN_GO_GRPC_VERSION="v1.3.0"

# protoc-gen-grpc-gateway is used to generate a reverse-proxy server that
# translates a RESTful HTTP API into gRPC.
echo "Installing protoc-gen-grpc-gateway version $GRPC_GATEWAY_VERSION..."
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@$GRPC_GATEWAY_VERSION

# protoc-gen-openapiv2 is used to generate OpenAPI v2 (formerly Swagger)
# documentation for the gRPC services. It's used to create API documentation
# that can be consumed by tools like Swagger UI.
echo "Installing protoc-gen-openapiv2 version $OPENAPIV2_VERSION..."
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@$OPENAPIV2_VERSION

# protoc-gen-go is used to generate Go code from the proto files. This
# is necessary for creating the data structures and serialization/
# deserialization code in Go.
echo "Installing protoc-gen-go version $PROTOC_GEN_GO_VERSION..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOC_GEN_GO_VERSION

# protoc-gen-go-grpc is used to generate Go code specifically for gRPC services
# from the .proto files. This includes server and client code for the gRPC
# services.
echo "Installing protoc-gen-go-grpc version $PROTOC_GEN_GO_GRPC_VERSION..."
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$PROTOC_GEN_GO_GRPC_VERSION

echo "Protobuf plugins installed successfully."
