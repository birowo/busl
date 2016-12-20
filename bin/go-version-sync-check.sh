#!/bin/bash

travis=$(cat .travis.yml | grep '^go:' | sed 's/^go: \([0-9]*\.[0-9]*\)\(\.[0-9]*\)\{0,1\}$/\1/')
goversion=$(< vendor/vendor.json jq -r '.heroku.goVersion' | sed 's/^go\(.*\)$/\1/')
diff <(echo $travis) <(echo $goversion)
