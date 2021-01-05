#!/bin/bash
docker build --file docker/rocketpool-dockerfile --tag kidkal/rocketpool:metrics .
pushd rocketpool-cli
go build
popd
cp -f rocketpool-cli/rocketpool-cli ~/bin/rocketpool-metrics
