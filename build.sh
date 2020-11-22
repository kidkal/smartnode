#!/bin/bash
docker build --file docker/rocketpool-dockerfile --tag local/rocketpool:latest .
pushd rocketpool-cli
go build
popd
cp -f rocketpool-cli/rocketpool-cli ~/bin/rocketpool