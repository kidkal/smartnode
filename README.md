# ROCKETPOOL - SMARTNODE

##### -=# Metrics Edition #=-

Use at own risk!  Original rocketpool project https://github.com/rocket-pool/smartnode

***Note*** running metrics consume additional CPU (especially whole network data like the leaderboard) so don't run this on machines with resource limitations.

---


## Installation Instructions
***Danger*** Proceed with caution. Do *not* go around downloading random stuff from random github repositories and expect your machine to remain in one piece.  The code will work of course, but be cautious when doing stuff like this!

Follow the usual rocketpool installation process:
https://github.com/rocket-pool/smartnode-install

Then run the commands:

```bash
wget https://github.com/kidkal/smartnode/releases/download/v0.0.1/metrics.tar.bz2
tar xfvj metrics.tar.bz2 --directory ~/.rocketpool
cd ~/.rocketpool/metrics
./rocketpool-metrics service start
```

This will:

- download the prebuilt executable with various configuration files
- extract the files to `~/.rocketpool/metrics`
- run the metrics docker containers:
	- `metrics_api`
	- `metrics_prometheus`
	- `metrics_grafana`

### Optional
- Enable metrics for geth node
- Enable metrics for beacon node
- Enable metrics for validator


## Metrics
Available at `http://metrics_api:2112/metrics` for your own scraping purposes or promql

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

## Build Instructions
Install the prerequisite development tools like `git`, `go` and `docker`

```sh
git clone https://github.com/kidkal/smartnode.git
cd smartnode
git checkout metrics
./build.sh
```

This will:

- build the `rocketpool-cli` binary and copy to `~/bin/rocketpool-metrics`
- build the `rocketpool` service binary, wrap it into a docker image with label `kidkal/rocketpool:metrics`


## Change description
- add almost all data that can be scraped from rocketpool contracts
- include histogram of scores
- include nodes with no validators
- show summary of minipool status
- add cli `rocketpool-metrics minipool leader` : spits out a list rocketpool minipools and their running profit/loss in csv
- add cli `rocketpool-metrics node leader` : spits out a list rocketpool nodes and the running profit/loss of its top 2 minipools in csv
- add metrics end point at `http://metrics_api:2112/metrics` for prometheus scraping


## To do
- fix ethClient.dial connect: cannot assign requested address issue
