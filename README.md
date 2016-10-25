# <img src="https://i.cloudup.com/WSKggRp4ZX.svg" width=78 /> <br/> busl [![Build Status](https://travis-ci.org/heroku/busl.svg?branch=master)](https://travis-ci.org/heroku/busl)


busl - the bustle part of hustle.

a simple pubsub service that runs on Heroku.

## Usage

### Creating streams

create a stream:

```
$ export STREAM_ID=$(curl http://localhost:5001/streams -X POST)
# STREAM_ID=b7e586c8404b74e1805f5a9543bc516f
```

### Subscribe

connect a consumer using the stream id:

```
$ curl http://localhost:5001/streams/$STREAM_ID
...
```

#### Disconnections

Subscribers can sometimes get disconnected. If the instance is cycled or deployed for example.
When this happens, you can specify a `Range` header specifying which where your streaming stopped.

```
$ curl http://localhost:5001/streams/$STREAM_ID -H "Range: bytes=100-"
```

SSE connections also handle the `Last-Event-ID` header.


### Publish
in a separate terminal, produce some data using the same stream id...

```
$ curl -H "Transfer-Encoding: chunked" http://localhost:5001/streams/$STREAM_ID -X POST
```

...and you see the busl.

## Setup

to setup to test and run busl, setup [godep](http://godoc.org/github.com/tools/godep)
and then:

```sh
$ make setup
```

## Test

to run tests:

```sh
$ make test
```

## Run

to run the server:

```sh
$ make web
```

## Deploy

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

## Docker setup

```sh
# Start
$ docker-compose start

# Grab the host / port combination chosen by docker
$ export URL=$(docker-compose port web 5000)

# Check health status
$ curl $URL/health
OK
```

## Busltee

The busltee command allows streaming a command's logs to a busl stream.

### Building

```sh
make busltee
```
