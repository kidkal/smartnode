package metrics

import (
    "time"
    "strconv"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/urfave/cli"

    "github.com/rocket-pool/rocketpool-go/node"
    "github.com/rocket-pool/rocketpool-go/rocketpool"
    "github.com/rocket-pool/rocketpool-go/utils/eth"
    apiNode "github.com/rocket-pool/smartnode/rocketpool/api/node"
    "github.com/rocket-pool/smartnode/shared/services"
    "github.com/rocket-pool/smartnode/shared/services/beacon"
    "github.com/rocket-pool/smartnode/shared/utils/hex"
    "github.com/rocket-pool/smartnode/shared/utils/log"
)


const (
    NodeDetailsBatchSize = 10
    TopMinipoolCount = 2
)


// RP metrics process
type RocketPoolMetrics struct {
    nodeScores             *prometheus.GaugeVec
    nodeMinipoolCounts     *prometheus.GaugeVec
    totalNodes             prometheus.Gauge
    activeNodes            prometheus.Gauge
}


type metricsProcess struct {
    rp *rocketpool.RocketPool
    bc beacon.Client
    metrics RocketPoolMetrics
    logger log.ColorLogger
}


func newMetricsProcss(c *cli.Context, logger log.ColorLogger) (*metricsProcess, error) {

    logger.Println("Enter newMetricsProcss")

    // Get services
    if err := services.RequireRocketStorage(c); err != nil { return nil, err }
    if err := services.RequireBeaconClientSynced(c); err != nil { return nil, err }
    rp, err := services.GetRocketPool(c)
    if err != nil { return nil, err }
    bc, err := services.GetBeaconClient(c)
    if err != nil { return nil, err }

    // Initialise metrics
    metrics := RocketPoolMetrics {
        nodeScores:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                //Subsystem:  "rocketpool",
                Name:       "node_score_eth",
                Help:       "sum of rewards/penalties of the top two minipools for this node",
            },
            []string{"address", "rank"},
        ),
        nodeMinipoolCounts:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                //Subsystem:  "rocketpool",
                Name:       "node_minipool_count",
                Help:       "number of activated minipools running for this node",
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
    }

    p := &metricsProcess {
        rp: rp,
        bc: bc,
        metrics: metrics,
        logger: logger,
    }

    logger.Println("Exit newMetricsProcss")
    return p, nil
}


// Start RP metrics process
func startMetricsProcess(p *metricsProcess) error {

    p.logger.Println("Enter startMetricsProcess")

    // Update metrics on interval
    err := updateMetrics(p)
    if err != nil {
        p.logger.Printlnf("Error in updateMetrics: %w", err)
    }
    updateMetricsTimer := time.NewTicker(metricsUpdateInterval)
    for _ = range updateMetricsTimer.C {
        err = updateMetrics(p)
        if err != nil {
            p.logger.Printlnf("Error in updateMetrics: %w", err)
        }
    }

    p.logger.Println("Exit startMetricsProcess")
    return nil
}


// Update node metrics
func updateMetrics(p *metricsProcess) error {
    p.logger.Println("Enter updateMetrics")

    nodeRanks, err := apiNode.GetNodeLeader(p.rp, p.bc)
    if err != nil { return err }

    for _, nodeRank := range nodeRanks {

        nodeAddress := hex.AddPrefix(nodeRank.Address.Hex())
        minipoolCount := len(nodeRank.Details)
        scoreEth := eth.WeiToEth(nodeRank.Score)

        // push into prometheus
        p.metrics.nodeScores.With(prometheus.Labels{"address":nodeAddress, "rank":strconv.Itoa(nodeRank.Rank)}).Set(scoreEth)
        p.metrics.nodeMinipoolCounts.With(prometheus.Labels{"address":nodeAddress}).Set(float64(minipoolCount))
    }

    nodeCount, err := node.GetNodeCount(p.rp, nil)
    if err != nil { return err }

    // Update node metrics
    p.metrics.totalNodes.Set(float64(nodeCount))
    p.metrics.activeNodes.Set(float64(len(nodeRanks)))

    p.logger.Println("Exit updateMetrics")
    return nil
}
