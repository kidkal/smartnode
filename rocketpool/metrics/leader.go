package metrics

import (
    "fmt"
    "math/big"
    "sort"
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/urfave/cli"
    "golang.org/x/sync/errgroup"
    "go.uber.org/multierr"

    "github.com/rocket-pool/rocketpool-go/deposit"
    "github.com/rocket-pool/rocketpool-go/minipool"
    "github.com/rocket-pool/rocketpool-go/node"
    "github.com/rocket-pool/rocketpool-go/network"
    "github.com/rocket-pool/rocketpool-go/rocketpool"
    "github.com/rocket-pool/rocketpool-go/settings"
    "github.com/rocket-pool/rocketpool-go/types"
    "github.com/rocket-pool/rocketpool-go/utils/eth"
    apiMinipool "github.com/rocket-pool/smartnode/rocketpool/api/minipool"
    apiNetwork "github.com/rocket-pool/smartnode/rocketpool/api/network"
    apiNode "github.com/rocket-pool/smartnode/rocketpool/api/node"
    "github.com/rocket-pool/smartnode/shared/services"
    "github.com/rocket-pool/smartnode/shared/services/beacon"
    "github.com/rocket-pool/smartnode/shared/types/api"
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
    minipoolCounts         *prometheus.GaugeVec
    minipoolBalance        *prometheus.GaugeVec
    minipoolQueue          *prometheus.GaugeVec
    networkFees            *prometheus.GaugeVec
    networkBlock           prometheus.Gauge
    networkBalances        *prometheus.GaugeVec
    settingsFlags          *prometheus.GaugeVec
}


type metricsProcess struct {
    rp *rocketpool.RocketPool
    bc beacon.Client
    account accounts.Account
    metrics RocketPoolMetrics
    logger log.ColorLogger
}


type networkStuff struct {
    Block uint64
    TotalETH *big.Int
    StakingETH *big.Int
    TotalRETH *big.Int
    DepositBalance *big.Int
    DepositExcessBalance *big.Int
    WithdrawBalance *big.Int
}


func newMetricsProcss(c *cli.Context, logger log.ColorLogger) (*metricsProcess, error) {

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
            []string{"address", "trusted", "timezone"},
        ),
        totalNodes:         promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            Subsystem:      "node",
            Name:           "total_count",
            Help:           "total number of nodes in Rocket Pool",
        }),
        minipoolCounts: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "minipool",
                Name:       "count",
                Help:       "minipools counts with various aggregations",
            },
            []string{"status"},
        ),
        minipoolBalance:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "minipool",
                Name:       "balance_eth",
                Help:       "balance of validator",
            },
            []string{"address", "validatorPubkey"},
        ),
        minipoolQueue:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "minipool",
                Name:       "queue_count",
                Help:       "number of minipools in queue for assignment",
            },
            []string{"depositType"},
        ),
        networkFees:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "network",
                Name:       "fee_rate",
                Help:       "network fees as rate of amount staked",
            },
            []string{"range"},
        ),
        networkBlock:       promauto.NewGauge(prometheus.GaugeOpts{
            Namespace:      "rocketpool",
            Subsystem:      "network",
            Name:           "updated_block",
            Help:           "block of lastest submitted balances",
        }),
        networkBalances:    promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "network",
                Name:       "balance_eth",
                Help:       "network balances and supplies in given unit",
            },
            []string{"unit"},
        ),
        settingsFlags:      promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace:  "rocketpool",
                Subsystem:  "settings",
                Name:       "flags_bool",
                Help:       "settings flags on rocketpool contracts",
            },
            []string{"flag"},
        ),
    }

    p := &metricsProcess {
        rp: rp,
        bc: bc,
        account: account,
        metrics: metrics,
        logger: logger,
    }

    return p, nil
}


// Start RP metrics process
func startMetricsProcess(p *metricsProcess) {

    // Update metrics on interval
    err := updateMetrics(p)
    if err != nil {
        p.logger.Printlnf("Error in updateMetrics: %w", err)
    }
    updateMetricsTimer := time.NewTicker(metricsUpdateInterval)
    for _ = range updateMetricsTimer.C {
        err = updateMetrics(p)
        if err != nil {
            // print error here instead of exit
            p.logger.Printlnf("Error in updateMetrics: %w", err)
        }
    }
}


// Update node metrics
func updateMetrics(p *metricsProcess) error {
    p.logger.Println("Enter updateMetrics")

    err1 := updateNodeCounts(p)
    err2 := updateMinipool(p)
    err3 := updateLeader(p)
    err4 := updateNetwork(p)
    err5 := updateMinipoolQueue(p)
    err6 := updateSettings(p)
    err := multierr.Combine(err1, err2, err3, err4, err5, err6)

    p.logger.Printlnf("Exit updateMetrics with %d errors", len(multierr.Errors(err)))
    return err
}


func updateNodeCounts(p *metricsProcess) error {

    nodeCount, err := node.GetNodeCount(p.rp, nil)
    if err != nil { return err }

    // Update node metrics
    p.metrics.totalNodes.Set(float64(nodeCount))

    return nil
}


func updateMinipool(p *metricsProcess) error {

    minipools, err := apiMinipool.GetNodeMinipoolDetails(p.rp, p.bc, p.account.Address)
    if err != nil { return err }

    for _, minipool := range minipools {
        address := hex.AddPrefix(minipool.Node.Address.Hex())
        validatorPubkey := hex.AddPrefix(minipool.ValidatorPubkey.Hex())
        balance := eth.WeiToEth(minipool.Validator.Balance)

        p.metrics.minipoolBalance.With(prometheus.Labels{"address":address, "validatorPubkey":validatorPubkey}).Set(balance)
    }

    return nil
}


func updateLeader(p *metricsProcess) error {

    nodeRanks, err := apiNode.GetNodeLeader(p.rp, p.bc)
    if err != nil { return err }

    updateScore(p, nodeRanks)
    updateHistogram(p, nodeRanks)
    updateNodeMinipoolCount(p, nodeRanks)
    updateMinipoolCount(p, nodeRanks)

    return nil
}


func updateScore(p *metricsProcess, nodeRanks []api.NodeRank) {
    p.metrics.nodeScores.Reset()

    for _, nodeRank := range nodeRanks {

        nodeAddress := hex.AddPrefix(nodeRank.Address.Hex())

        if nodeRank.Score != nil {
            scoreEth := eth.WeiToEth(nodeRank.Score)
            p.metrics.nodeScores.With(prometheus.Labels{"address":nodeAddress, "rank":strconv.Itoa(nodeRank.Rank)}).Set(scoreEth)
        }
    }
}


func updateHistogram(p *metricsProcess, nodeRanks []api.NodeRank) {
    p.metrics.nodeScoreHist.Reset()

    histogram := make(map[float64]int, 100)
    var sumScores float64

    for _, nodeRank := range nodeRanks {

        if nodeRank.Score != nil {
            scoreEth := eth.WeiToEth(nodeRank.Score)

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
}


func updateNodeMinipoolCount(p *metricsProcess, nodeRanks []api.NodeRank) {
    p.metrics.nodeMinipoolCounts.Reset()

    for _, nodeRank := range nodeRanks {

        nodeAddress := hex.AddPrefix(nodeRank.Address.Hex())
        minipoolCount := len(nodeRank.Details)
        labels := prometheus.Labels {
            "address":nodeAddress,
            "trusted":strconv.FormatBool(nodeRank.Trusted),
            "timezone":nodeRank.TimezoneLocation,
        }
        p.metrics.nodeMinipoolCounts.With(labels).Set(float64(minipoolCount))
    }
}


func updateMinipoolCount(p *metricsProcess, nodeRanks []api.NodeRank) {
    p.metrics.minipoolCounts.Reset()

    var totalCount, initializedCount, prelaunchCount, stakingCount, withdrawableCount, dissolvedCount int
    var validatorExistsCount, validatorActiveCount int

    for _, nodeRank := range nodeRanks {
        totalCount += len(nodeRank.Details)
        for _, minipool := range nodeRank.Details {
            switch minipool.Status.Status {
                case types.Initialized:  initializedCount++
                case types.Prelaunch:    prelaunchCount++
                case types.Staking:      stakingCount++
                case types.Withdrawable: withdrawableCount++
                case types.Dissolved:    dissolvedCount++
        	}
            if minipool.Validator.Exists { validatorExistsCount ++ }
            if minipool.Validator.Active { validatorActiveCount ++ }
        }
    }
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"total"}).Set(float64(totalCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"initialized"}).Set(float64(initializedCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"prelaunch"}).Set(float64(prelaunchCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"staking"}).Set(float64(stakingCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"withdrawable"}).Set(float64(withdrawableCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"dissolved"}).Set(float64(dissolvedCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"validatorExists"}).Set(float64(validatorExistsCount))
    p.metrics.minipoolCounts.With(prometheus.Labels{"status":"validatorActive"}).Set(float64(validatorActiveCount))
}


func updateNetwork(p *metricsProcess) error {

    nodeFees, err := apiNetwork.GetNodeFee(p.rp)
    if err != nil { return err }

    p.metrics.networkFees.With(prometheus.Labels{"range":"current"}).Set(nodeFees.NodeFee)
    p.metrics.networkFees.With(prometheus.Labels{"range":"min"}).Set(nodeFees.MinNodeFee)
    p.metrics.networkFees.With(prometheus.Labels{"range":"target"}).Set(nodeFees.TargetNodeFee)
    p.metrics.networkFees.With(prometheus.Labels{"range":"max"}).Set(nodeFees.MaxNodeFee)

    stuff, err := getOtherNetworkStuff(p.rp)
    if err != nil { return err }

    p.metrics.networkBlock.Set(float64(stuff.Block))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"TotalETH"}).Set(eth.WeiToEth(stuff.TotalETH))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"StakingETH"}).Set(eth.WeiToEth(stuff.StakingETH))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"TotalRETH"}).Set(eth.WeiToEth(stuff.TotalRETH))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"Deposit"}).Set(eth.WeiToEth(stuff.DepositBalance))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"DepositExcess"}).Set(eth.WeiToEth(stuff.DepositExcessBalance))
    p.metrics.networkBalances.With(prometheus.Labels{"unit":"Withdraw"}).Set(eth.WeiToEth(stuff.WithdrawBalance))

    return nil
}


func getOtherNetworkStuff(rp *rocketpool.RocketPool) (*networkStuff, error) {
    stuff := networkStuff{}

    // Sync
    var wg errgroup.Group

    // Get data
    wg.Go(func() error {
        block, err := network.GetBalancesBlock(rp, nil)
        if err == nil {
            stuff.Block = block
        }
        return err
    })
    wg.Go(func() error {
        totalETH, err := network.GetTotalETHBalance(rp, nil)
        if err == nil {
            stuff.TotalETH = totalETH
        }
        return err
    })
    wg.Go(func() error {
        stakingETH, err := network.GetStakingETHBalance(rp, nil)
        if err == nil {
            stuff.StakingETH = stakingETH
        }
        return err
    })
    wg.Go(func() error {
        totalRETH, err := network.GetTotalRETHSupply(rp, nil)
        if err == nil {
            stuff.TotalRETH = totalRETH
        }
        return err
    })
    wg.Go(func() error {
        depositBalance, err := deposit.GetBalance(rp, nil)
        if err == nil {
            stuff.DepositBalance = depositBalance
        }
        return err
    })
    wg.Go(func() error {
        depositBalance, err := deposit.GetBalance(rp, nil)
        if err == nil {
            stuff.DepositBalance = depositBalance
        }
        return err
    })
    wg.Go(func() error {
        depositExcessBalance, err := deposit.GetExcessBalance(rp, nil)
        if err == nil {
            stuff.DepositExcessBalance = depositExcessBalance
        }
        return err
    })
    wg.Go(func() error {
        withdrawBalance, err := network.GetWithdrawalBalance(rp, nil)
        if err == nil {
            stuff.WithdrawBalance = withdrawBalance
        }
        return err
    })

    // Wait for data
    if err := wg.Wait(); err != nil {
        return nil, err
    }

    // Return response
    return &stuff, nil
}


func updateMinipoolQueue(p *metricsProcess) error {
    var wg errgroup.Group
    var fullQueueLength, halfQueueLength, emptyQueueLength uint64

    // Get data
    wg.Go(func() error {
        response, err := minipool.GetQueueLength(p.rp, types.Full, nil)
        if err == nil {
            fullQueueLength = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := minipool.GetQueueLength(p.rp, types.Half, nil)
        if err == nil {
            halfQueueLength = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := minipool.GetQueueLength(p.rp, types.Empty, nil)
        if err == nil {
            emptyQueueLength = response
        }
        return err
    })

    // Wait for data
    if err := wg.Wait(); err != nil {
        return err
    }
    p.metrics.minipoolQueue.With(prometheus.Labels{"depositType":"Full"}).Set(float64(fullQueueLength))
    p.metrics.minipoolQueue.With(prometheus.Labels{"depositType":"Half"}).Set(float64(halfQueueLength))
    p.metrics.minipoolQueue.With(prometheus.Labels{"depositType":"Empty"}).Set(float64(emptyQueueLength))

    return nil
}

func updateSettings(p *metricsProcess) error {
    var wg errgroup.Group
    var depositEnabled, assignDepositEnabled, minipoolWithdrawEnabled, submitBalancesEnabled, processWithdrawalEnabled, nodeRegistrationEnabled, nodeDepositEnabled bool

    // Get data
    wg.Go(func() error {
        response, err := settings.GetDepositEnabled(p.rp, nil)
        if err == nil {
            depositEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetAssignDepositsEnabled(p.rp, nil)
        if err == nil {
            assignDepositEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetMinipoolSubmitWithdrawableEnabled(p.rp, nil)
        if err == nil {
            minipoolWithdrawEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetSubmitBalancesEnabled(p.rp, nil)
        if err == nil {
            submitBalancesEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetProcessWithdrawalsEnabled(p.rp, nil)
        if err == nil {
            processWithdrawalEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetNodeRegistrationEnabled(p.rp, nil)
        if err == nil {
            nodeRegistrationEnabled = response
        }
        return err
    })
    wg.Go(func() error {
        response, err := settings.GetNodeDepositEnabled(p.rp, nil)
        if err == nil {
            nodeDepositEnabled = response
        }
        return err
    })

    // Wait for data
    if err := wg.Wait(); err != nil {
        return err
    }
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"DepositEnabled"}).Set(float64(B2i(depositEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"AssignDepositEnabled"}).Set(float64(B2i(assignDepositEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"MinipoolWithdrawEnabled"}).Set(float64(B2i(minipoolWithdrawEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"SubmitBalancesEnabled"}).Set(float64(B2i(submitBalancesEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"ProcessWithdrawalEnabled"}).Set(float64(B2i(processWithdrawalEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"NodeRegistrationEnabled"}).Set(float64(B2i(nodeRegistrationEnabled)))
    p.metrics.settingsFlags.With(prometheus.Labels{"flag":"NodeDepositEnabled"}).Set(float64(B2i(nodeDepositEnabled)))

    return nil
}

func B2i(b bool) int8 {
    if b {
        return 1
    }
    return 0
}
