package cmd

import (
	"fmt"

	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var verbose bool
var logger = loggerPkg.GetLogger()

func newRootCmd(version string, buildTime string, gitCommit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simple-api-gateway",
		Short: "simple-api-gateway is a tool to serve as an API gateway, which can be used to proxy requests to multiple backends",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logger.SetVerbose(true)
			} else {
				logger.SetVerbose(false)
			}
			logger.Reset()
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	cmd.AddCommand(newVersionCmd(version, buildTime, gitCommit))
	cmd.AddCommand(newCheckCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newGenCmd())
	return cmd
}

func Execute(version string, buildTime string, gitCommit string) error {
	if err := newRootCmd(version, buildTime, gitCommit).Execute(); err != nil {
		logger.Error("error executing root command", zap.Error(err))
		return fmt.Errorf("error executing root command: %w", err)
	}

	return nil
}
