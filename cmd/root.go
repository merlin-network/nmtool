package cmd

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/merlin-network/nemo/app"
	"github.com/spf13/cobra"

	"github.com/merlin-network/nmtool/cmd/testnet"
)

var furyGrpcUrl string

var rootCmd = &cobra.Command{
	Use:   "nmtool",
	Short: "Dev tools for working with the nemo blockchain.",
}

// Execute runs the root command.
func Execute() error {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)
	config.Seal()

	var cdc *codec.LegacyAmino = app.MakeEncodingConfig().Amino

	rootCmd.AddCommand(EstimateBlockHeightCmd())
	rootCmd.AddCommand(InflationRootCmd())
	rootCmd.AddCommand(MaccAddrCmd())
	rootCmd.AddCommand(NodeKeysCmd(cdc))
	rootCmd.AddCommand(SwapIDCmd(cdc))
	rootCmd.AddCommand(testnet.Cmd())

	return rootCmd.Execute()
}
