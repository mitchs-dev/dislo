#!/bin/bash

if [[ $protoFile == "" ]]; then
  protoFile="dislo.proto"
fi

# Make sure script is run from the proto directory
currentDir=${PWD##*/}
if [ $currentDir != "proto" ]; then
  echo "Please run this script from the proto directory"
  exit 1
fi

appDir="../pkg"

rm -rf $appDir/generated
mkdir -p $appDir

# Generate code for all targets
protoc \
  -I. \
  --go_out=$appDir \
  --go-grpc_out=$appDir \
  --proto_path=. \
  $protoFile || exit 1

# Generate descriptor specifically for grpcurl
protoc \
  --proto_path=. \
  --include_imports \
  --include_source_info \
  -o services.desc \
  $protoFile || exit 1
