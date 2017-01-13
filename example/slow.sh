#!/bin/bash
STREAM_ID=$(uuidgen)
URL=http://$BUSL_HOST/streams/$STREAM_ID

echo "Creating a stream"
curl $URL -X PUT

echo "Publishing to the stream"
(go run cmd/busltee/main.go $URL -- ./example/compile 60 > log/command.log) &

echo "Listening $i"
curl $URL
