package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/merlin-network/nemo/x/bep3/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/merlin-network/nmtool/binance"
)

var (
	furyDeputiesStrings map[string]string = map[string]string{
		"bnb":  "fury1r4v2zdhdalfj2ydazallqvrus9fkphmgnfwgej",
		"btcb": "fury14qsmvzprqvhwmgql9fr0u3zv9n2qla8zmdxxys",
		"busd": "fury1hh4x3a4suu5zyaeauvmv7ypf7w9llwlfsh0fe5",
		"xrpb": "fury1c0ju5vnwgpgxnrktfnkccuth9xqc68dcpllncc",
	}
	bnbDeputiesStrings map[string]string = map[string]string{
		"bnb":  "bnb1jh7uv2rm6339yue8k4mj9406k3509kr4wt5nxn",
		"btcb": "bnb1xz3xqf4p2ygrw9lhp5g5df4ep4nd20vsywnmpr",
		"busd": "bnb10zq89008gmedc6rrwzdfukjk94swynd7dl97w8",
		"xrpb": "bnb15jzuvvg2kf0fka3fl2c8rx0kc3g6wkmvsqhgnh",
	}
)

// SwapIDCmd returns a command to calculate a bep3 swap ID for binance and nemo chains.
func SwapIDCmd(cdc *codec.LegacyAmino) *cobra.Command {

	furyDeputies := map[string]sdk.AccAddress{}
	for k, v := range furyDeputiesStrings {
		furyDeputies[k] = mustFuryAccAddressFromBech32(v)
	}
	bnbDeputies := map[string]binance.AccAddress{}
	for k, v := range bnbDeputiesStrings {
		bnbDeputies[k] = mustBnbAccAddressFromBech32(v)
	}

	cmd := &cobra.Command{
		Use:   "swap-id random_number_hash original_sender_address deputy_addres_or_denom",
		Short: "Calculate binance and nemo swap IDs given swap details.",
		Long: fmt.Sprintf(`A swap's ID is: hash(swap.RandomNumberHash, swap.Sender, swap.SenderOtherChain)
One of the senders is always the deputy's address, the other is the user who initiated the first swap (the original sender).
Corresponding swaps on each chain have the same RandomNumberHash, but switched address order.

The deputy can be one of %v to automatically use the mainnet deputy addresses, or an arbitrary address.
The original sender and deputy address cannot be from the same chain.
`, getKeys(furyDeputiesStrings)),
		Example: "swap-id 464105c245199d02a4289475b8b231f3f73918b6f0fdad898825186950d46f36 bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x busd",
		Args:    cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {

			randomNumberHash, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}

			// try and decode the bech32 address as either nemo or bnb
			addressFury, errFury := sdk.AccAddressFromBech32(args[1])
			addressBnb, errBnb := binance.AccAddressFromBech32(args[1])

			// fail if both decoding failed
			isFuryAddress := errFury == nil && errBnb != nil
			isBnbAddress := errFury != nil && errBnb == nil
			if !isFuryAddress && !isBnbAddress {
				return fmt.Errorf("can't unmarshal original sender address as either nemo or bnb: (%s) (%s)", errFury.Error(), errBnb.Error())
			}

			// calculate swap IDs
			depArg := args[2]
			var swapIDFury, swapIDBnb []byte
			if isFuryAddress {
				// check sender isn't a deputy
				for _, dep := range furyDeputies {
					if addressFury.Equals(dep) {
						return fmt.Errorf("original sender address cannot be deputy address: %s", dep)
					}
				}
				// pick deputy address
				var bnbDeputy binance.AccAddress
				bnbDeputy, ok := bnbDeputies[depArg]
				if !ok {
					bnbDeputy, err = binance.AccAddressFromBech32(depArg)
					if err != nil {
						return fmt.Errorf("can't unmarshal deputy address as bnb address (%s)", err)
					}
				}
				// calc ids
				swapIDFury = types.CalculateSwapID(randomNumberHash, addressFury, bnbDeputy.String())
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, bnbDeputy, addressFury.String())
			} else {
				// check sender isn't a deputy
				for _, dep := range bnbDeputies {
					if bytes.Equal(addressBnb, dep) {
						return fmt.Errorf("original sender address cannot be deputy address %s", dep)
					}
				}
				// pick deputy address
				var furyDeputy sdk.AccAddress
				furyDeputy, ok := furyDeputies[depArg]
				if !ok {
					furyDeputy, err = sdk.AccAddressFromBech32(depArg)
					if err != nil {
						return fmt.Errorf("can't unmarshal deputy address as nemo address (%s)", err)
					}
				}
				// calc ids
				swapIDBnb = binance.CalculateSwapID(randomNumberHash, addressBnb, furyDeputy.String())
				swapIDFury = types.CalculateSwapID(randomNumberHash, furyDeputy, addressBnb.String())
			}

			outString, err := formatResults(swapIDFury, swapIDBnb)
			if err != nil {
				return err
			}
			fmt.Println(outString)
			return nil
		},
	}

	return cmd
}

func formatResults(swapIDFury, swapIDBnb []byte) (string, error) {
	result := struct {
		FurySwapID string `yaml:"fury_swap_id"`
		BnbSwapID  string `yaml:"bnb_swap_id"`
	}{
		FurySwapID: hex.EncodeToString(swapIDFury),
		BnbSwapID:  hex.EncodeToString(swapIDBnb),
	}
	bz, err := yaml.Marshal(result)
	return string(bz), err
}

func mustFuryAccAddressFromBech32(address string) sdk.AccAddress {
	a, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return a
}

func mustBnbAccAddressFromBech32(address string) binance.AccAddress {
	a, err := binance.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return a
}

func getKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
