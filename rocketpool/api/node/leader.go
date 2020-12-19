package node

import (
    "math/big"
    "sort"

    "github.com/ethereum/go-ethereum/common"
    "github.com/urfave/cli"

    "github.com/rocket-pool/rocketpool-go/node"
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
    nodeAddresses, err := node.GetNodeAddresses(rp, nil)
    if err != nil { return nil, err }

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

    nodeRanks := make([]api.NodeRank, len(nodeAddresses))
    i := 0

    for address, vals := range nodeToValMap {
        nodeRanks[i].Address = address
        nodeRanks[i].Details = vals
        nodeRanks[i].Score = calculateNodeScore(vals)
        i++
    }

    sort.SliceStable(nodeRanks[0:i], func(m, n int) bool { return nodeRanks[m].Score.Cmp(nodeRanks[n].Score) > 0 })
    for k := 0; k < i; k++ {
        nodeRanks[k].Rank = k + 1
    }

    // add nodes with no validators
    for _, address := range nodeAddresses {
        if _, ok := nodeToValMap[address]; !ok {
            nodeRanks[i].Address = address
            nodeRanks[i].Rank = 999999999
            i++
        }
    }

    return nodeRanks, nil
}


func calculateNodeScore(vals []api.MinipoolDetails) *big.Int {
    // score formula: take the top N performing validators
    // sum up their profits or losses
    // profit is defined as: current balance - initial node deposit - user deposit
    // unless something is broken, this should be current balance - 32
    // unit is wei

    var prevMax *big.Int
    score := new(big.Int)

    // remove non-existing validators from scoring
    // use selection sort so we don't need to alloc more memory
    for j := 0; j < TopMinipoolCount && j < len(vals); j++ {
        var currMax *api.MinipoolDetails
        for k := 0; k < len(vals); k++ {
            if vals[k].Validator.Exists &&
                vals[k].Validator.Balance != nil &&
                (currMax == nil || vals[k].Validator.Balance.Cmp(currMax.Validator.Balance) > 0) &&
                (prevMax == nil || vals[k].Validator.Balance.Cmp(prevMax) < 0) {
                    currMax = &vals[k]
            }
        }

        if currMax == nil {
            break
        }

        score.Add(score, currMax.Validator.Balance)
        score.Sub(score, currMax.Node.DepositBalance)
        score.Sub(score, currMax.User.DepositBalance)

        prevMax = currMax.Validator.Balance
    }

    return score
}
