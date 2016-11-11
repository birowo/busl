#!/bin/bash
REDIS_URL=redis://localhost:6379
PORT=6000

STREAM_ID=$(uuidgen)
URL=http://busl-development.$HEROKU_CLOUD.herokuappdev.com/streams/$STREAM_ID

echo "Creating a stream"
curl $URL -X PUT

echo "Publishing to the stream"
(go run cmd/busltee/main.go $URL -- ./example/compile > log/command.log) &

(
  for i in {1..2}; do
    sleep 20
    echo "Restarting busl $i"
    heroku restart -a busl-development
  done
) &

for i in {1..3}; do
  echo "Listening $i"
  curl $URL
  sleep 5
done

sleep 60
