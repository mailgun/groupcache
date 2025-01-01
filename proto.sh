#! /bin/sh

# Make sure the script fails fast.
set -e
set -u
set -x

PROTO_DIR=groupcachepb

protoc -I=$PROTO_DIR \
    --go_out=$PROTO_DIR \
    $PROTO_DIR/groupcache.proto

protoc -I=$PROTO_DIR \
   --go_out=. \
    $PROTO_DIR/example.proto && mv ./example.pb.go ./example_pb_test.go
