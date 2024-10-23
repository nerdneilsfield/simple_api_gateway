package cmd

import (
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newGenCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "gen",
		Short:        "generate the api gateway config",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("generate the api gateway config", zap.String("config", args[0]))
			err := config.GenerateExampleConfigPath(args[0])
			if err != nil {
				logger.Error("failed to generate the api gateway config", zap.Error(err))
				return err
			}
			return nil
		},
	}
}
