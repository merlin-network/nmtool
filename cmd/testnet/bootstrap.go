package testnet

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/merlin-network/nmtool/config/generate"
	"github.com/spf13/cobra"
)

// this env variable is used in supported nemo templates to allow override of the image tag
// automated chain upgrades make use of it to switch between binary versions.
const furyTagEnv = "FURY_TAG"

func BootstrapCmd() *cobra.Command {
	bootstrapCmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "A convenience command that creates a nemo testnet with the input configTemplate (defaults to master)",
		Long: `Generate a nemo testnet and optionally run/integrate various other resources.

# General Overview
This command runs local networks by performing the following three steps:
1. Configure the desired resources with templates that are combined into a primary docker-compose configuration.
2. Standup the docker-compose containers and network.
3. Perform various modifications, restarts, configurations to the live networks based on the desired topology.

# Templates
The building blocks of the bootstrap command's services are templates which are defined in the
directory nmtool/config/templates. As of right now, the template you run Fury with is configurable
at runtime via the 'nemo.configTemplate' flag. The Fury templates contain nemo config directories
(including a genesis.json) that are supported with the corresponding nemo docker image tag.

Some templates, like "master", support overriding the image tag via the FURY_TAG env variable.

# IBC
The bootstrap command supports running a secondary chain and opening an IBC channel between the
primary Fury node and the secondary chain. To set this up, simply use the --ibc flag.

Once the two chains are started, the necessary txs are run to open a channel between the two chains
and a relayer is started to relay transactions between them. The primary denom of the secondary chain
is "uatom" and it runs under the docker container named "ibcchain".

# Automated Chain Upgrades
The bootstrap command supports running a chain that is then upgraded via an upgrade handler. The following
flags are all required to run an automated software upgrade:

--upgrade-name           - the name of the registered upgrade handler to be run.
--upgrade-height         - the height at which the upgrade should occur.
--upgrade-base-image-tag - the docker image tag of Fury that with which the chain is started.

Note that the upgrade height must be high enough to facilitate the submission of an upgrade proposal,
and the voting on it. If used with --ibc, note that the upgrade is initiated _after_ the IBC channel
is opened, which can take 70+ blocks.

When these flags are defined, the chain is initially started with the --upgrade-base-image-tag tag.
As soon as the chain is configured & producing blocks, a committee proposal is submitted to update
the chain. The committee uses First-Pass-the-Post voting so passes as soon as it gets consensus.
The committee member account votes on the proposal and then we wait for the upgrade height to be
reached. At that point, the chain halts and is restarted with the updated image tag.`,
		Example: `Run nemo node with particular template:
$ nmtool testnet bootstrap --nemo.configTemplate v0.12

Run nemo & another chain with open IBC channel & relayer:
$ nmtool testnet bootstrap --ibc

Run nemo & an ethereum node:
$ nmtool testnet bootstrap --geth

The master template supports dynamic override of the Fury node's container image:
$ FURY_TAG=v0.21.0 nmtool testnet bootstrap

Test a chain upgrade from v0.19.2 -> v0.21.0:
$ FURY_TAG=v0.21.0 nmtool testnet bootstrap --upgrade-name v0.21.0 --upgrade-height 15 --upgrade-base-image-tag v0.19.2
`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateBootstrapFlags(); err != nil {
				return err
			}

			// shutdown existing networks if a docker-compose.yaml already exists.
			if _, err := os.Stat(generatedPath("docker-compose.yaml")); err == nil {
				if err2 := dockerComposeCmd("down").Run(); err2 != nil {
					return err2
				}
			}

			// remove entire generated dir in order to start from scratch
			if err := os.RemoveAll(generatedConfigDir); err != nil {
				return fmt.Errorf("could not clear old generated config: %v", err)
			}

			// generate nemo node configuration
			if err := generate.GenerateFuryConfig(furyConfigTemplate, generatedConfigDir); err != nil {
				return err
			}
			// handle ibc configuration
			if ibcFlag {
				if err := generate.GenerateIbcConfigs(generatedConfigDir); err != nil {
					return err
				}
			}
			// handle geth configuration
			if gethFlag {
				if err := generate.GenerateGethConfig(generatedConfigDir); err != nil {
					return err
				}
			}

			// pull the nemo image tag if not overridden to be "local"
			furyTagOverride := os.Getenv(furyTagEnv)
			if furyTagOverride != "local" {
				if err := dockerComposeCmd("pull").Run(); err != nil {
					fmt.Println(err.Error())
				}
			}

			upCmd := dockerComposeCmd("up", "-d")
			// when doing automated chain upgrade, ensure the node starts with the desired image tag
			// if this is empty, the docker-compose should default to intended image tag
			if chainUpgradeBaseImageTag != "" {
				upCmd.Env = os.Environ()
				upCmd.Env = append(upCmd.Env, fmt.Sprintf("%s=%s", furyTagEnv, chainUpgradeBaseImageTag))
				fmt.Printf("starting chain with image tag %s\n", chainUpgradeBaseImageTag)
			}
			if err := upCmd.Run(); err != nil {
				fmt.Println(err.Error())
			}

			if ibcFlag {
				if err := setupIbcChannelAndRelayer(); err != nil {
					return err
				}
			}

			// validation of all necessary data for an automated chain upgrade is performed in validateBootstrapFlags()
			if chainUpgradeName != "" {
				if err := runChainUpgrade(); err != nil {
					return err
				}
			}

			return nil
		},
	}

	bootstrapCmd.Flags().StringVar(&furyConfigTemplate, "nemo.configTemplate", "master", "the directory name of the template used to generating the nemo config")
	bootstrapCmd.Flags().BoolVar(&ibcFlag, "ibc", false, "flag for if ibc is enabled")
	bootstrapCmd.Flags().BoolVar(&gethFlag, "geth", false, "flag for if geth is enabled")

	// optional data for running an automated chain upgrade
	bootstrapCmd.Flags().StringVar(&chainUpgradeName, "upgrade-name", "", "name of automated chain upgrade to run, if desired. the upgrade must be defined in the nemo image container.")
	bootstrapCmd.Flags().Int64Var(&chainUpgradeHeight, "upgrade-height", 0, "height of automated chain upgrade to run.")
	bootstrapCmd.Flags().StringVar(&chainUpgradeBaseImageTag, "upgrade-base-image-tag", "", "the nemo docker image tag that will be upgraded.\nthe chain is initialized from this tag and then upgraded to the new image.\nthe binary must be compatible with the nemo.configTemplate genesis.json.")

	return bootstrapCmd
}

func validateBootstrapFlags() error {
	hasUpgradeName := chainUpgradeName != ""
	hasUpgradeBaseImageTag := chainUpgradeBaseImageTag != ""
	// the upgrade flags are all or nothing. both the upgrade name and the image tag are required for
	// an automated chain upgrade
	if (hasUpgradeName && !hasUpgradeBaseImageTag) || (hasUpgradeBaseImageTag && !hasUpgradeName) {
		return fmt.Errorf("automated chain upgrades require both --upgrade-name and --upgrade-base-image-tag to be defined")
	}
	// if running an automate chain upgrade, there must be a sufficiently high upgrade height.
	if hasUpgradeName && chainUpgradeHeight < 10 {
		// TODO: is 10 a sufficient height for an upgrade to occur with proposal & voting? probs not..
		return fmt.Errorf("upgrade height must be > 10, found %d", chainUpgradeHeight)
	}
	return nil
}

func setupIbcChannelAndRelayer() error {
	// wait for chain to be up and running before setting up ibc
	if err := waitForBlock(1, 5*time.Second); err != nil {
		return err
	}

	fmt.Printf("Starting ibc connection between chains...\n")
	// add new channel to relayer config
	setupIbcPathCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", generatedPath("relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "paths", "new", furyChainId, ibcChainId, "transfer")
	setupIbcPathCmd.Stdout = os.Stdout
	setupIbcPathCmd.Stderr = os.Stderr
	if err := setupIbcPathCmd.Run(); err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("[relayer] failed to setup ibc path")
	}
	// open the channel between nemo and ibcnode
	openConnectionCmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s:%s", generatedPath("relayer"), "/home/relayer/.relayer"), "--network", "generated_default", relayerImageTag, "rly", "transact", "link", "transfer")
	openConnectionCmd.Stdout = os.Stdout
	openConnectionCmd.Stderr = os.Stderr
	if err := openConnectionCmd.Run(); err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("[relayer] failed to open ibc connection")
	}
	fmt.Printf("IBC connection complete, starting relayer process...\n")
	// setup and run the relayer
	if err := generate.AddRelayerToNetwork(generatedConfigDir); err != nil {
		return err
	}
	if err := dockerComposeCmd("up", "-d", "relayer").Run(); err != nil {
		return err
	}
	// prune temp containers used to initialize ibc channel
	pruneCmd := exec.Command("docker", "container", "prune", "-f")
	pruneCmd.Stdout = os.Stdout
	pruneCmd.Stderr = os.Stderr
	if err := pruneCmd.Run(); err != nil {
		return err
	}
	fmt.Println("IBC relayer ready!")
	return nil
}

func runChainUpgrade() error {
	fmt.Printf(
		"configured for automated chain upgrade\n\tupgrade name: %s\n\tupgrade height: %d\n\tstarting tag: %s\n",
		chainUpgradeName, chainUpgradeHeight, chainUpgradeBaseImageTag,
	)

	// write upgrade proposal to json file
	upgradeJson, err := writeUpgradeProposal()
	if err != nil {
		return err
	}

	// wait for chain to start producing blocks
	if err := waitForBlock(1, 5*time.Second); err != nil {
		return err
	}

	// submit upgrade proposal via God Committee (committee 3)
	fmt.Println("submitting upgrade proposal")
	cmd := fmt.Sprintf("tx committee submit-proposal 3 %s --gas auto --gas-adjustment 1.2 --gas-prices 0.05ufury --from committee -y",
		upgradeJson,
	)
	if err := runFuryCli(strings.Split(cmd, " ")...); err != nil {
		return err
	}

	// vote on the committee proposal
	cmd = "tx committee vote 1 yes --from committee --gas auto --gas-adjustment 1.2 --gas-prices 0.01ufury -y"
	if err := runFuryCli(strings.Split(cmd, " ")...); err != nil {
		return err
	}

	// wait for chain halt at upgrade height
	if err := waitForBlock(chainUpgradeHeight, time.Duration(chainUpgradeHeight)*4*time.Second); err != nil {
		return err
	}
	fmt.Printf("chain has reached upgrade height (%d) and halted!\n", chainUpgradeHeight)

	fmt.Println("restarting chain with upgraded image")
	// this runs with the desired image because FURY_TAG will be correctly set, or if that is unset,
	// the docker-compose files supporting upgrades default to the desired template version.
	if err := dockerComposeCmd("up", "--force-recreate", "-d", "furynode").Run(); err != nil {
		return err
	}

	return nil
}

// writeUpgradeProposal writes a proposal json to a file in the furynode container and returns the path
func writeUpgradeProposal() (string, error) {
	content := fmt.Sprintf(`{
		"@type": "/cosmos.upgrade.v1beta1.SoftwareUpgradeProposal",
		"title": "Automated Chain Upgrade",
		"description": "An auto-magical chain upgrade performed by nmtool.",
		"plan": { "name": "%s", "height": "%d" }
	}`, chainUpgradeName, chainUpgradeHeight)
	// write the file to a location inside the container
	return "/root/.nemo/config/upgrade-proposal.json", os.WriteFile(
		generatedPath("nemo", "initstate", ".nemo", "config", "upgrade-proposal.json"),
		[]byte(content),
		0644,
	)
}

func waitForBlock(n int64, timeout time.Duration) error {
	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 2 * time.Second
	b.MaxElapsedTime = timeout
	return backoff.Retry(blockGTE(n), b)
}

// blockGTE is a backoff operation that uses nemo's CLI to query the chain for the current block number
// the operation fails in the following cases:
// 1. the chain cannot be reached, 2. result cannot be parsed, 3. current height is less than desired height `n`
func blockGTE(n int64) backoff.Operation {
	return func() error {
		cmd := "nemo status | jq -r .sync_info.latest_block_height"
		out, err := exec.Command("docker-compose", "-f", generatedPath("docker-compose.yaml"), "exec", "-T", "furynode", "bash", "-c", cmd).Output()
		if err != nil {
			return err
		}
		height, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return err
		}
		if height < n {
			fmt.Printf("waiting for chain to reach height %d, currently @ %d\n", n, height)
			return fmt.Errorf("waiting for height %d, found %d", n, height)
		}
		return nil
	}
}

// runFuryCli execs into the nemo container and runs `nemo args...`
func runFuryCli(args ...string) error {
	pieces := make([]string, 4, len(args)+4)
	pieces[0] = "exec"
	pieces[1] = "-T"
	pieces[2] = "furynode"
	pieces[3] = "nemo"
	pieces = append(pieces, args...)
	return dockerComposeCmd(pieces...).Run()
}

func dockerComposeCmd(args ...string) *exec.Cmd {
	pieces := make([]string, 2, len(args)+2)
	pieces[0] = "--file"
	pieces[1] = generatedPath("docker-compose.yaml")
	pieces = append(pieces, args...)
	cmd := exec.Command("docker-compose", pieces...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}
