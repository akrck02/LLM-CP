#!/bin/sh
# Generates the Go code of hub.proto. protoc is downloaded into out/ (ignored by git) at the
# version below; the Go plugins are installed with go install at theirs.
set -eu

protoc_version="33.2"
protoc_gen_go_version="v1.36.11"
protoc_gen_go_grpc_version="v1.6.0"

protocol_directory="$(cd "$(dirname "$0")" && pwd)"
tools_directory="$protocol_directory/../out/tools"
mkdir -p "$tools_directory"

if [ ! -x "$tools_directory/bin/protoc" ]; then
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64) platform="linux-x86_64" ;;
    Linux-aarch64) platform="linux-aarch_64" ;;
    Darwin-x86_64) platform="osx-x86_64" ;;
    Darwin-arm64) platform="osx-aarch_64" ;;
    *) echo "protocol/generate.sh: no protoc for $(uname -s)-$(uname -m)" >&2; exit 1 ;;
  esac
  # TODO(akrck02): check the archive against a pinned sha256.
  archive="protoc-$protoc_version-$platform.zip"
  curl -sSL -o "$tools_directory/$archive" \
    "https://github.com/protocolbuffers/protobuf/releases/download/v$protoc_version/$archive"
  unzip -oq "$tools_directory/$archive" -d "$tools_directory"
fi

GOBIN="$tools_directory/bin" go install "google.golang.org/protobuf/cmd/protoc-gen-go@$protoc_gen_go_version"
GOBIN="$tools_directory/bin" go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@$protoc_gen_go_grpc_version"

PATH="$tools_directory/bin:$PATH" protoc \
  --proto_path="$protocol_directory" \
  --go_out="$protocol_directory" --go_opt=paths=source_relative \
  --go-grpc_out="$protocol_directory" --go-grpc_opt=paths=source_relative \
  "$protocol_directory/hub.proto"
