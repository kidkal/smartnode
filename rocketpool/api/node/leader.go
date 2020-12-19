package node

import (
    "math/big"
    "sort"

    "github.com/ethereum/go-ethereum/common"
    "github.com/urfave/cli"

    //"github.com/rocket-pool/rocketpool-go/node"
    "github.com/rocket-pool/rocketpool-go/rocketpool"
    "github.com/rocket-pool/smartnode/rocketpool/api/minipool"
    "github.com/rocket-pool/smartnode/shared/services"
    "github.com/rocket-pool/smartnode/shared/services/beacon"
    "github.com/rocket-pool/smartnode/shared/types/api"
)

// Settings
const (
    NodeDetailsBatchSize = 10
    TopMinipoolCount = 2
)


func getLeader(c *cli.Context) (*api.NodeLeaderResponse, error) {
    // Get services
    if err := services.RequireRocketStorage(c); err != nil { return nil, err }
    if err := services.RequireBeaconClientSynced(c); err != nil { return nil, err }
    rp, err := services.GetRocketPool(c)
    if err != nil { return nil, err }
    bc, err := services.GetBeaconClient(c)
    if err != nil { return nil, err }

    // Response
    response := api.NodeLeaderResponse{}

    nodeRanks, err := GetNodeLeader(rp, bc)
    if err != nil { return nil, err }

    response.Nodes = nodeRanks
    return &response, nil
}


func GetNodeLeader(rp *rocketpool.RocketPool, bc beacon.Client) ([]api.NodeRank, error) {

    minipools, err := minipool.GetAllMinipoolDetails(rp, bc)
    if err != nil { return nil, err }
    //nodeAddresses, err := node.GetNodeAddresses(rp, nil)
    //if err != nil { return nil, err }

    // Get stating and has validator minipools
    // put minipools into map by address
    nodeToValMap := make(map[common.Address][]api.MinipoolDetails, len(minipools))
    for _, minipool := range minipools {
        // Add to status list
        address := minipool.Node.Address
        if _, ok := nodeToValMap[address]; !ok {
            nodeToValMap[address] = []api.MinipoolDetails{}
        }
        nodeToValMap[address] = append(nodeToValMap[address], minipool)
    }

    nodeRanks := make([]api.NodeRank, len(nodeToValMap))
    i := 0

    for address, vals := range nodeToValMap {
        nodeRanks[i].Address = address
        nodeRanks[i].Details = vals
        sort.SliceStable(vals, func(i, j int) bool { return vals[i].Validator.Balance.Cmp(vals[j].Validator.Balance) > 0 })
        count := TopMinipoolCount
        if count > len(vals) { count = len(vals) }

        // score formula: take the top N performing validators
        // sum up their profits or losses
        // profit is defined as: current balance - initial node deposit - user deposit
        // unless something is broken, this should be current balance - 32
        // unit is wei

        nodeRanks[i].Score = new(big.Int)
        for j := 0; j < count; j++ {
            nodeRanks[i].Score.Add(nodeRanks[i].Score, vals[j].Validator.Balance)
            nodeRanks[i].Score.Sub(nodeRanks[i].Score, vals[j].Node.DepositBalance)
            nodeRanks[i].Score.Sub(nodeRanks[i].Score, vals[j].User.DepositBalance)
        }
        i++
    }

    sort.SliceStable(nodeRanks[0:i], func(m, n int) bool { return nodeRanks[m].Score.Cmp(nodeRanks[n].Score) > 0 })
    for k := 0; k < i; k++ {
        nodeRanks[k].Rank = k + 1
    }

    // add nodes with no validators
    // for _, address := range nodeAddresses {
    //     if _, ok := nodeToValMap[address]; !ok {
    //         nodeRanks[i].Address = address
    //         nodeRanks[i].Score = new(big.Int)
    //         nodeRanks[i].Rank = 999999999
    //         i++
    //     }
    // }

    return nodeRanks, nil
}
