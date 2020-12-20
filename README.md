# WARNING *darrr be dragonsss*!

Use at own risk!  This is a hack of a very nice project https://github.com/rocket-pool/smartnode

---

## Build instructions
- follow the usual rocketpool installation steps
- build with `./build.sh` this will:
    - build the `rocketpool` cli and put into your `~/bin`
    - build the `rocketpool` docker image with label `local/rocketpool:latest`
- replace the files in `~/.rocketpool` with the ones in `./config-sample`


## Change description
- include histogram of scores
- include nodes with no validators
- show summary of minipool status
- add cli `rocketpool minipool leader` - spits out a list rocketpool minipools and their running profit/loss in csv
- add cli `rocketpool node leader` - spits out a list rocketpool nodes and the running profit/loss of its top 2 minipools in csv
- add metrics end point at `http://metrics:2112/metrics` for scraping by prometheus

## To do
- fix ethClient.dial connect: cannot assign requested address issue
- some kind of consolidation with node and api containers
- moar data points!
