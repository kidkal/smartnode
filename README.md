# ROCKETPOOL - SMARTNODE

##### -=# Metrics Edition #=-

Use at own risk!  Original rocketpool project https://github.com/rocket-pool/smartnode

***Note*** running metrics consume additional CPU (especially whole network data like the leaderboard) so don't run this on machines with resource restrictions.

---

## Build Instructions
Install the prerequisite development tools like `git`, `go` and `docker`

```sh
> git clone https://github.com/kidkal/smartnode.git
> cd smartnode
> git checkout metrics
> ./build.sh
```


This will:

- build the `rocketpool-cli` binary and copy to `~/bin/rocketpool-metrics`
- build the `rocketpool` service binary, wrap it into a docker image with label `kidkal/rocketpool:metrics`


## Install Instructions
- follow the usual rocketpool installation process:
https://github.com/rocket-pool/smartnode-install

TODO - yet to be implemented, intention is to:

- download metrics release
- extract into location *near* rocketpool eg `~/.rocketpool-metrics`
- docker-compose build
- docker-compose up -d

- (old) replace the files in `~/.rocketpool` with the ones in `./config-sample`

## Metrics
Available at `http://metrics:2112/metrics`

```
rocketpool_minipool_count{status}
rocketpool_minipool_queue_count{depositType}
rocketpool_network_balance_eth{unit}
rocketpool_network_fee_rate{range}
rocketpool_network_updated_block
rocketpool_node_minipool_count{address,timezone,trusted}
rocketpool_node_score_eth{address,rank}
rocketpool_node_score_hist_eth{le}
rocketpool_node_score_hist_eth_count
rocketpool_node_score_hist_eth_sum
rocketpool_node_total_count
rocketpool_settings_flags_bool{flag="AssignDepositEnabled"}
rocketpool_settings_flags_bool{flag="DepositEnabled"}
rocketpool_settings_flags_bool{flag="MinipoolWithdrawEnabled"}
rocketpool_settings_flags_bool{flag="NodeDepositEnabled"}
rocketpool_settings_flags_bool{flag="NodeRegistrationEnabled"}
rocketpool_settings_flags_bool{flag="ProcessWithdrawalEnabled"}
rocketpool_settings_flags_bool{flag="SubmitBalancesEnabled"}
rocketpool_settings_maximum_pool_eth
rocketpool_settings_minimum_deposit_eth
```

## Change description
- added almost all data that can be scraped from rocketpool contracts
- include histogram of scores
- include nodes with no validators
- show summary of minipool status
- add cli `rocketpool minipool leader` - spits out a list rocketpool minipools and their running profit/loss in csv
- add cli `rocketpool node leader` - spits out a list rocketpool nodes and the running profit/loss of its top 2 minipools in csv
- add metrics end point at `http://metrics:2112/metrics` for prometheus scraping


## To do
- fix ethClient.dial connect: cannot assign requested address issue
- some kind of consolidation with node and api containers
- better user experience (prebuilt binary and container)
