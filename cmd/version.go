package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(version string, buildTime string, gitCommit string) *cobra.Command {
	return &cobra.Command{
		Use:          "version",
		Short:        "simple-api-gateway version",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			slogan := `
 _______  _______ _________   _______  _______ _________ _______           _______          
(  ___  )(  ____ )\__   __/  (  ____ \(  ___  )\__   __/(  ____ \|\     /|(  ___  )|\     /|
| (   ) || (    )|   ) (     | (    \/| (   ) |   ) (   | (    \/| )   ( || (   ) |( \   / )
| (___) || (____)|   | |     | |      | (___) |   | |   | (__    | | _ | || (___) | \ (_) / 
|  ___  ||  _____)   | |     | | ____ |  ___  |   | |   |  __)   | |( )| ||  ___  |  \   /  
| (   ) || (         | |     | | \_  )| (   ) |   | |   | (      | || || || (   ) |   ) (   
| )   ( || )      ___) (___  | (___) || )   ( |   | |   | (____/\| () () || )   ( |   | |   
|/     \||/       \_______/  (_______)|/     \|   )_(   (_______/(_______)|/     \|   \_/   
                                                                                            
				`
			fmt.Fprintln(cmd.OutOrStdout(), slogan)
			fmt.Fprintln(cmd.OutOrStdout(), "Author: dengqi935@gmail.com")
			fmt.Fprintln(cmd.OutOrStdout(), "Github: https://github.com/nerdneilsfield/simple-api-gateway")
			fmt.Fprintln(cmd.OutOrStdout(), "Wiki: https://nerdneilsfield.github.io/simple-api-gateway/")
			fmt.Fprintf(cmd.OutOrStdout(), "simple-api-gateway: %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "buildTime: %s\n", buildTime)
			fmt.Fprintf(cmd.OutOrStdout(), "gitCommit: %s\n", gitCommit)
			fmt.Fprintf(cmd.OutOrStdout(), "goVersion: %s\n", runtime.Version())
		},
	}
}
