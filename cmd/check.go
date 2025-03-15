package cmd

import (
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "check",
		Short:        "check that the api gateway config is valid",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("check the api gateway config", zap.String("config", args[0]))
			config_, err := config.ParseConfig(args[0])
			if err != nil {
				return err
			}
			return config.ValidateConfig(config_)
		},
	}
}
