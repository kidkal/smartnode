package metrics

import (
    "fmt"
    "sort"
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/urfave/cli"

    "github.com/rocket-pool/rocketpool-go/node"
    "github.com/rocket-pool/rocketpool-go/rocketpool"
    "github.com/rocket-pool/rocketpool-go/utils/eth"
    "github.com/rocket-pool/smartnode/rocketpool/api/minipool"
    apiNode "github.com/rocket-pool/smartnode/rocketpool/api/node"
    "github.com/rocket-pool/smartnode/shared/services"
    "github.com/rocket-pool/smartnode/shared/services/beacon"
    "github.com/rocket-pool/smartnode/shared/utils/hex"
    "github.com/rocket-pool/smartnode/shared/utils/log"
)


const (
    BucketInterval = 0.025
)


// RP metrics process
type RocketPoolMetrics struct {
    nodeScores             *prometheus.GaugeVec
    nodeScoreHist          *prometheus.GaugeVec
    nodeScoreHistSum       prometheus.Gauge
    nodeScoreHistCount     prometheus.Gauge
    nodeMinipoolCounts     *prometheus.GaugeVec
    totalNodes             prometheus.Gauge
    minipoolBalance        *prometheus.GaugeVec
}


type metricsProcess struct {
    rp *rocketpool.RocketPool
    bc beacon.Client
    account accounts.Account
    metrics RocketPoolMetrics
    logger log.ColorLogger
}


func newMetricsProcss(c *cli.Context, logger log.ColorLogger) (*metricsProcess, error) {

    logger.Println("Enter newMetricsProcss")

    // Get services
    if err := services.RequireRocketStorage(c); err != nil { return nil, err }
    if err := services.RequireBeaconClientSynced(c); err != nil { return nil, err }
    w, err := services.GetWallet(c)
    if err != nil { return nil, err }
    rp, err := services.GetRocketPool(c)
    if err != nil { return nil, err }
    bc, err := services.GetBeaconClient(c)
    if err != nil { return nil, err }
    account, err := w.GetNodeAccount()
    if err != nil { return nil, err }

    // Initialise metrics
    metrics := RocketPoolMetrics {
        nodeScores:         promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "node",
                Name:       "score_eth",
                Help:       "sum of rewards/penalties of the top two minipools for this node",
            },
            []string{"address", "rank"},
        ),
        nodeScoreHist: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "node",
                Name:       "score_hist_eth",
                Help:       "distribution of sum of rewards/penalties of the top two minipools in rocketpool network",
                },
            []string{"le"},
        ),
        nodeScoreHistSum:   promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            Subsystem:      "node",
            Name:           "score_hist_eth_sum",
        }),
        nodeScoreHistCount: promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            Subsystem:      "node",
            Name:           "score_hist_eth_count",
        }),
        nodeMinipoolCounts: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "node",
                Name:       "minipool_count",
                Help:       "number of activated minipools running for this node",
            },
            []string{"address"},
        ),
        totalNodes:         promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            Subsystem:      "node",
            Name:           "total_count",
            Help:           "total number of nodes in Rocket Pool",
        }),
        minipoolBalance:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "minipool",
                Name:       "balance_eth",
                Help:       "balance of validator",
            },
            []string{"address", "validatorPubkey"},
        ),
    }

    p := &metricsProcess {
        rp: rp,
        bc: bc,
        account: account,
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

    var err error
    err = updateNodeCounts(p)
    err = updateMinipool(p)
    err = updateLeader(p)

    p.logger.Println("Exit updateMetrics")
    return err
}


func updateNodeCounts(p *metricsProcess) error {
    p.logger.Println("Enter updateNodeCounts")

    nodeCount, err := node.GetNodeCount(p.rp, nil)
    if err != nil { return err }

    // Update node metrics
    p.metrics.totalNodes.Set(float64(nodeCount))

    p.logger.Println("Exit updateNodeCounts")
    return nil
}


func updateMinipool(p *metricsProcess) error {
    p.logger.Println("Enter updateMinipool")

    minipools, err := minipool.GetNodeMinipoolDetails(p.rp, p.bc, p.account.Address)
    if err != nil { return err }

    for _, minipool := range minipools {
        address := hex.AddPrefix(minipool.Node.Address.Hex())
        validatorPubkey := hex.AddPrefix(minipool.ValidatorPubkey.Hex())
        balance := eth.WeiToEth(minipool.Validator.Balance)

        p.metrics.minipoolBalance.With(prometheus.Labels{"address":address, "validatorPubkey":validatorPubkey}).Set(balance)
    }

    p.logger.Println("Exit updateMinipool")
    return nil
}


func updateLeader(p *metricsProcess) error {
    p.logger.Println("Enter updateLeader")

    nodeRanks, err := apiNode.GetNodeLeader(p.rp, p.bc)
    if err != nil { return err }

    p.metrics.nodeScores.Reset()
    p.metrics.nodeMinipoolCounts.Reset()
    p.metrics.nodeScoreHist.Reset()

    histogram := make(map[float64]int, 100)
    var sumScores float64

    for _, nodeRank := range nodeRanks {

        // push into prometheus
        nodeAddress := hex.AddPrefix(nodeRank.Address.Hex())
        minipoolCount := len(nodeRank.Details)
        p.metrics.nodeMinipoolCounts.With(prometheus.Labels{"address":nodeAddress}).Set(float64(minipoolCount))

        if nodeRank.Score != nil {
            scoreEth := eth.WeiToEth(nodeRank.Score)
            p.metrics.nodeScores.With(prometheus.Labels{"address":nodeAddress, "rank":strconv.Itoa(nodeRank.Rank)}).Set(scoreEth)

            // find next highest bucket to put in
            bucket := float64(int(scoreEth / BucketInterval)) * BucketInterval
        	if (bucket < scoreEth) {
        	    bucket = bucket + BucketInterval
        	}
            if _, ok := histogram[bucket]; !ok {
                histogram[bucket] = 0
            }
            histogram[bucket]++
            sumScores += scoreEth
        }
    }

    buckets := make([]float64, 0, len(histogram))
    for b := range histogram {
        buckets = append(buckets, b)
    }
    sort.Float64s(buckets)

    accCount := 0
    for _, b := range buckets {
        accCount += histogram[b]
        p.metrics.nodeScoreHist.With(prometheus.Labels{"le":fmt.Sprintf("%.3f", b)}).Set(float64(accCount))
    }

    p.metrics.nodeScoreHistSum.Set(sumScores)
    p.metrics.nodeScoreHistCount.Set(float64(accCount))

    p.logger.Println("Exit updateLeader")
    return nil
}
