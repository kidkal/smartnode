package metrics

import (
    "net/http"
    "time"
    "fmt"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/urfave/cli"
    "golang.org/x/sync/errgroup"
)


// Config
const UPDATE_METRICS_INTERVAL string = "15s"
var updateMetricsInterval, _ = time.ParseDuration(UPDATE_METRICS_INTERVAL)


// Register metrics command
func RegisterCommands(app *cli.App, name string, aliases []string) {
    app.Commands = append(app.Commands, cli.Command{
        Name:      name,
        Aliases:   aliases,
        Usage:     "Run Rocket Pool metrics daemon",
        Action: func(c *cli.Context) error {
            return run(c)
        },
    })
}


// Run process
func run(c *cli.Context) error {
    fmt.Println("enter run")
    var wg1 errgroup.Group

    // Start metrics processes
    wg1.Go(func() error {
        err := startRocketPoolMetricsProcess(c)
        return err
    })

    // Serve metrics
    wg1.Go(func() error {
        http.Handle("/metrics", promhttp.Handler())
        return http.ListenAndServe(":2112", nil)
    })

    if err := wg1.Wait(); err != nil {
        return err
    }

    return nil
}
