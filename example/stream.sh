#!/bin/bash
APP_NAME=damien-busl
STREAM_ID=$(uuidgen)
URL=http://$APP_NAME.herokuapp.com/streams/$STREAM_ID

echo "Creating a stream"
curl $URL -X PUT

echo "Publishing to the stream"
(go run cmd/busltee/main.go $URL -- ./example/compile > log/command.log) &

echo "Listening $i"
curl $URL

sleep 60
