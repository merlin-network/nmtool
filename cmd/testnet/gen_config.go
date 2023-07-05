package testnet

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fanfury-sports/nmtool/config/generate"
)

func GenConfigCmd() *cobra.Command {
	genConfigCmd := &cobra.Command{
		Use:   "gen-config services_to_include...",
		Short: "Generate a complete docker-compose configuration for a new testnet.",
		Long: fmt.Sprintf(`Generate a docker-compose.yaml file and any other necessary config files needed by services.

available services: %s
`, supportedServices),
		Example:   "gen-config nemo binance deputy --nemo.configTemplate v0.10",
		ValidArgs: supportedServices,
		Args:      cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		RunE: func(_ *cobra.Command, args []string) error {

			// 1) clear out generated config folder
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// 2) generate a complete docker-compose config
			if stringSlice(args).contains(furyServiceName) {
				if err := generate.GenerateFuryConfig(furyConfigTemplate, generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(binanceServiceName) {
				if err := generate.GenerateBnbConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if stringSlice(args).contains(deputyServiceName) {
				if err := generate.GenerateDeputyConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if ibcFlag {
				if err := generate.GenerateIbcChainConfig(generatedConfigDir); err != nil {
					return err
				}
			}
			if gethFlag {
				if err := generate.GenerateGethConfig(generatedConfigDir); err != nil {
					return err
				}
			}

			return nil
		},
	}

	genConfigCmd.Flags().StringVar(&furyConfigTemplate, "nemo.configTemplate", "master", "the directory name of the template used to generating the nemo config")
	genConfigCmd.Flags().BoolVar(&ibcFlag, "ibc", false, "flag for if ibc is enabled")
	genConfigCmd.Flags().BoolVar(&gethFlag, "geth", false, "flag for if geth node is enabled")

	return genConfigCmd
}
