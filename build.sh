#!/bin/bash
rm -f metrics.tar.bz2
rm -rf metrics

docker build --file docker/rocketpool-dockerfile --tag kidkal/rocketpool:metrics .
pushd rocketpool-cli
go build
popd

mkdir metrics
cp -r config/* metrics
rm -r metrics/chains
cp rocketpool-cli/rocketpool-cli metrics/rocketpool-metrics
tar cfvj metrics.tar.bz2 metrics
