GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

travis: tidy test

test: .PHONY govendor
	env $$(cat .env) govendor test +local -race

setup: hooks tidy
	cp .env.sample .env

hooks:
	ln -fs ../../bin/git-pre-commit.sh .git/hooks/pre-commit

precommit: tidy test

tidy: goimports govendor
	./bin/go-version-sync-check.sh
	test -z "$$(goimports -l -d $(GO_FILES) | tee /dev/stderr)"
	govendor vet +local

web: .PHONY
	env $$(cat .env) go run cmd/busl/main.go

goimports:
	go get golang.org/x/tools/cmd/goimports

govendor:
	go get github.com/kardianos/govendor

busltee: .PHONY bin/busltee

bin/busltee: .PHONY
	docker build -t heroku/busl:latest .
	docker run --rm -i heroku/busl:latest tar cz bin/busltee | tar x

.PHONY:
