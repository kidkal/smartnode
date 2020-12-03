package node

import (
    "fmt"

    "github.com/rocket-pool/rocketpool-go/utils/eth"
    "github.com/urfave/cli"

    "github.com/rocket-pool/smartnode/shared/services/rocketpool"
    "github.com/rocket-pool/smartnode/shared/utils/hex"
)


func getLeader(c *cli.Context) error {

    // Get RP client
    rp, err := rocketpool.NewClientFromCtx(c)
    if err != nil { return err }
    defer rp.Close()

    // Get node status
    response, err := rp.NodeLeader()
    if err != nil { return err }

    // Print & return
    if len(response.Nodes) == 0 {
        fmt.Println("No Rocketpool nodes")
        return nil
    }

    fmt.Printf("%d Rocketpool nodes\n", len(response.Nodes))
    fmt.Println("")
    fmt.Println("Rank,Node address,Score (ETH),Minipool count")

    for _, nodeRank := range response.Nodes {
        nodeAddress := hex.AddPrefix(nodeRank.Address.Hex())
        score := eth.WeiToEth(nodeRank.Score)
        fmt.Printf("%4d,%s,%+0.10f,%4d", nodeRank.Rank, nodeAddress, score, len(nodeRank.Details))
        fmt.Println("")
    }
    return nil
}
