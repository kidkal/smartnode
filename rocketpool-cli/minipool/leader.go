package minipool

import (
    "fmt"
    "sort"

    "github.com/rocket-pool/rocketpool-go/types"
    "github.com/rocket-pool/rocketpool-go/utils/eth"
    "github.com/urfave/cli"

    "github.com/rocket-pool/smartnode/shared/services/rocketpool"
    "github.com/rocket-pool/smartnode/shared/types/api"
    "github.com/rocket-pool/smartnode/shared/utils/hex"
)


func getLeader(c *cli.Context) error {

    // Get RP client
    rp, err := rocketpool.NewClientFromCtx(c)
    if err != nil { return err }
    defer rp.Close()

    // Get minipool statuses
    status, err := rp.MinipoolLeader()
    if err != nil {
        return err
    }

    // Get minipools by status
    minipools := []api.MinipoolDetails{}
    for _, minipool := range status.Minipools {

        // Add to status list
        if minipool.Status.Status == types.Staking && minipool.Validator.Exists {
            minipools = append(minipools, minipool)
        }
    }

    // Print & return
    if len(status.Minipools) == 0 {
        fmt.Println("No active minipools")
        return nil
    }

    sort.SliceStable(minipools, func(i, j int) bool { return eth.WeiToEth(minipools[i].Validator.Balance) > eth.WeiToEth(minipools[j].Validator.Balance) })

    fmt.Printf("%d active and staking minipools\n", len(minipools))
    fmt.Println("")
    fmt.Println("Rank,Node address,Validator pubkey,RP status update time,Accumulated reward/penalty (ETH)")

    for i, minipool := range minipools {
        nodeAddress := hex.AddPrefix(minipool.Node.Address.Hex())
        validatorAddress := hex.AddPrefix(minipool.ValidatorPubkey.Hex())
        statusTime := minipool.Status.StatusTime.Format("2006-01-02T15:04:05-0700")
        diffBalance := eth.WeiToEth(minipool.Validator.Balance) - eth.WeiToEth(minipool.Node.DepositBalance) - eth.WeiToEth(minipool.User.DepositBalance)
        fmt.Printf("%4d,%s,%s,%s,%+0.10f", i+1, nodeAddress, validatorAddress, statusTime, diffBalance)
        fmt.Println("")
    }
    return nil

}
