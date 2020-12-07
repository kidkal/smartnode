package metrics

import (
//    "bytes"
//    "errors"
    "fmt"
    "math/big"
    "sort"
    "time"
//    "log"
//    "os"

    "github.com/ethereum/go-ethereum/common"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/urfave/cli"

    "github.com/rocket-pool/rocketpool-go/node"
    "github.com/rocket-pool/rocketpool-go/rocketpool"
    "github.com/rocket-pool/rocketpool-go/types"
    "github.com/rocket-pool/rocketpool-go/utils/eth"
    "github.com/rocket-pool/smartnode/rocketpool/api/minipool"
    "github.com/rocket-pool/smartnode/shared/services"
    "github.com/rocket-pool/smartnode/shared/services/beacon"
    "github.com/rocket-pool/smartnode/shared/types/api"
    "github.com/rocket-pool/smartnode/shared/utils/hex"
)


const (
    NodeDetailsBatchSize = 10
    TopMinipoolCount = 2
)


// RP metrics process
type RocketPoolMetrics struct {
    leaderboard            *prometheus.GaugeVec
    totalNodes             prometheus.Gauge
    activeNodes            prometheus.Gauge
    activeMinipools        prometheus.Gauge
}


// Start RP metrics process
func startRocketPoolMetricsProcess(c *cli.Context) error {
    fmt.Println("enter startRocketPoolMetricsProcess")

    // Get services
    if err := services.RequireRocketStorage(c); err != nil { return err }
    if err := services.RequireBeaconClientSynced(c); err != nil { return err }
    rp, err := services.GetRocketPool(c)
    if err != nil { return err }
    bc, err := services.GetBeaconClient(c)
    if err != nil { return err }

    // Initialise metrics
    metrics := &RocketPoolMetrics {
        leaderboard:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                //Subsystem:  "rocketpool",
                Name:       "node_score_eth",
                Help:       "sum of rewards/penalties of the top two minipools for this node",
            },
            []string{"address"},
        ),
        totalNodes:         promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            //Subsystem:      "rocketpool",
            Name:           "node_total_count",
            Help:           "total number of nodes in Rocket Pool",
        }),
        activeNodes:        promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            //Subsystem:      "rocketpool",
            Name:           "node_active_count",
            Help:           "number of active nodes in Rocket Pool",
        }),
        activeMinipools:   promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            //Subsystem:      "rocketpool",
            Name:           "node_minipool_count",
            Help:           "number of active minipools in Rocket Pool",
        }),
    }

    fmt.Println("init finished")
    // Update metrics on interval
    err = updateNodeMetrics(rp, bc, metrics)
    if err != nil { return err }
    updateMetricsTimer := time.NewTicker(updateMetricsInterval)
    for _ = range updateMetricsTimer.C {
        err := updateNodeMetrics(rp, bc, metrics)
        if err != nil { return err }
    }

    return nil
}


// Update node metrics
func updateNodeMetrics(rp *rocketpool.RocketPool, bc beacon.Client, p *RocketPoolMetrics) error {
    fmt.Println("enter updateNodeMetrics")

    minipools, err := minipool.GetAllMinipoolDetails(rp, bc)
    if err != nil { return err }

    // Get minipools with staking status and existing validator
    // put minipools into map by node address
    nodeToValMap := make(map[common.Address][]api.MinipoolDetails, len(minipools))
    for _, minipool := range minipools {
        // Add to status list
        if minipool.Status.Status == types.Staking && minipool.Validator.Exists {
            address := minipool.Node.Address
            if _, ok := nodeToValMap[address]; !ok {
                nodeToValMap[address] = []api.MinipoolDetails{}
            }
            nodeToValMap[address] = append(nodeToValMap[address], minipool)
        }
    }

    for address, vals := range nodeToValMap {

        nodeAddress := hex.AddPrefix(address.Hex())

        sort.SliceStable(vals, func(i, j int) bool { return vals[i].Validator.Balance.Cmp(vals[j].Validator.Balance) > 0 })
        count := TopMinipoolCount
        if count > len(vals) { count = len(vals) }

        // score formula: take the top N performing minipools
        // sum up their profits or losses
        // profit is defined as: current balance - initial node deposit - user deposit
        // unless something is broken, this should be current balance - 32
        // unit is converted to eth

        score := new(big.Int)
        for j := 0; j < count; j++ {
            score.Add(score, vals[j].Validator.Balance)
            score.Sub(score, vals[j].Node.DepositBalance)
            score.Sub(score, vals[j].User.DepositBalance)
        }
        scoreEth := eth.WeiToEth(score)

        // push into prometheus
        p.leaderboard.With(prometheus.Labels{"address":nodeAddress}).Set(scoreEth)
    }

    nodeCount, err := node.GetNodeCount(rp, nil)
    if err != nil { return err }

    // Update node metrics
    p.totalNodes.Set(float64(nodeCount))
    p.activeNodes.Set(float64(len(nodeToValMap)))
    p.activeMinipools.Set(float64(len(minipools)))

    return nil
}
