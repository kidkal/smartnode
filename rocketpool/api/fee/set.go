package fee

import (
    "errors"
    "fmt"

    "github.com/urfave/cli"

    "github.com/rocket-pool/smartnode-cli/rocketpool/services/database"
)


// Set the target user fee to vote for
func setTargetUserFee(c *cli.Context, feePercent float64) error {

    // Initialise database
    db, err := database.NewDatabase(c.GlobalString("database"))
    if err != nil {
        return err
    }
    defer db.Close()

    // Set target user fee percent
    err = db.Put("user.fee.target", feePercent)
    if err != nil {
        return errors.New("Error setting target user fee percentage: " + err.Error())
    }

    // Log & return
    fmt.Println(fmt.Sprintf("Target user fee to vote for successfully set to %.2f%%", feePercent))
    return nil

}
