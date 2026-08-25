#!/bin/bash

# Get the directory where the script resides
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Define the full path to the task.proto file
PROTO_FILE="$SCRIPT_DIR/task.proto"

# Generate Go protobuf and gRPC code
protoc --proto_path="$SCRIPT_DIR" \
  --go_out="$SCRIPT_DIR" --go_opt=paths=source_relative \
  --go-grpc_out="$SCRIPT_DIR" --go-grpc_opt=paths=source_relative \
  "$(basename "$PROTO_FILE")"

echo "gRPC code generated successfully."