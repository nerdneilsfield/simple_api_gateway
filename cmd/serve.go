package cmd

import (
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/nerdneilsfield/simple_api_gateway/internal/router"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newServeCmd(gitCommit string) *cobra.Command {
	return &cobra.Command{
		Use:          "serve",
		Short:        "serve the api gateway with the given config",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("serve the api gateway", zap.String("config", args[0]))
			config_, err := config.ParseConfig(args[0])
			if err != nil {
				return err
			}
			if err := config.ValidateConfig(config_); err != nil {
				return err
			}
			router.Run(config_, gitCommit)
			return nil
		},
	}
}
