package metrics

import (
    "net/http"
    "time"

    "github.com/fatih/color"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/urfave/cli"

    "github.com/rocket-pool/smartnode/shared/utils/log"
)


// Config
const (
    MetricsColor = color.BgGreen
)
var metricsUpdateInterval, _ = time.ParseDuration("5m")

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
    logger := log.NewColorLogger(MetricsColor)
    logger.Println("Enter metrics.run")

    p, err := newMetricsProcss(c, logger)
    if err != nil {
        logger.Printlnf("Error in newMetricsProcss: %w", err)
        return err
    }

    // Start metrics processes
    go (func() { startMetricsProcess(p) })()

    // Serve metrics
    http.Handle("/metrics", promhttp.Handler())
    err = http.ListenAndServe(":2112", nil)

    logger.Printlnf("Exit metrics.run: %w", err)
    return err
}
