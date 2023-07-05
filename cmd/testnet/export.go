package testnet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func ExportCmd() *cobra.Command {
	exportCmd := &cobra.Command{
		Use:     "export",
		Short:   "Pauses the current nemo testnet, exports the current nemo testnet state to a JSON file, then restarts the testnet.",
		Example: "export",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cmd := exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "stop")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				return err
			}
			// docker ps -aqf "name=containername"
			furyContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_furynode_1")
			furyContainer, err := furyContainerIDCmd.Output()
			if err != nil {
				return err
			}

			ibcChainContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_ibcnode_1")
			ibcContainer, err := ibcChainContainerIDCmd.Output()
			if err != nil {
				return err
			}

			makeNewFuryImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(furyContainer)), "nemo-export-temp")

			furyImageOutput, err := makeNewFuryImageCmd.Output()
			if err != nil {
				return err
			}

			makeNewIbcImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(ibcContainer)), "ibc-export-temp")
			ibcImageOutput, err := makeNewIbcImageCmd.Output()
			if err != nil {
				return err
			}

			localFuryMountPath := generatedPath("nemo", "initstate", ".nemo", "config")
			localIbcMountPath := generatedPath("ibcchain", "initstate", ".nemo", "config")

			furyExportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.nemo/config", localFuryMountPath)),
				"nemo-export-temp",
				"nemo", "export")
			furyExportJSON, err := furyExportCmd.Output()
			if err != nil {
				return err
			}

			ibcExportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.nemo/config", localIbcMountPath)),
				"ibc-export-temp",
				"nemo", "export")
			ibcExportJSON, err := ibcExportCmd.Output()
			if err != nil {
				return err
			}
			ts := time.Now().Unix()
			furyFilename := fmt.Sprintf("nemo-export-%d.json", ts)
			ibcFilename := fmt.Sprintf("ibc-export-%d.json", ts)

			fmt.Printf("Created exports %s and %s\nCleaning up...", furyFilename, ibcFilename)

			err = os.WriteFile(furyFilename, furyExportJSON, 0644)
			if err != nil {
				return err
			}
			err = os.WriteFile(ibcFilename, ibcExportJSON, 0644)
			if err != nil {
				return err
			}

			// docker ps -aqf "name=containername"
			tempFuryContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=nemo-export-temp")
			tempFuryContainer, err := tempFuryContainerIDCmd.Output()
			if err != nil {
				return err
			}
			tempIbcContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=ibc-export-temp")
			tempIbcContainer, err := tempIbcContainerIDCmd.Output()
			if err != nil {
				return err
			}

			deleteFuryContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempFuryContainer)))
			err = deleteFuryContainerCmd.Run()
			if err != nil {
				return err
			}
			deleteIbcContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempIbcContainer)))
			err = deleteIbcContainerCmd.Run()
			if err != nil {
				return err
			}

			deleteFuryImageCmd := exec.Command("docker", "rmi", strings.TrimSpace(string(furyImageOutput)))
			err = deleteFuryImageCmd.Run()
			if err != nil {
				return err
			}
			deleteIbcImageCmd := exec.Command("docker", "rmi", strings.TrimSpace(string(ibcImageOutput)))
			err = deleteIbcImageCmd.Run()
			if err != nil {
				return err
			}

			fmt.Printf("Restarting testnet...")
			restartCmd := exec.Command("docker-compose", "--file", generatedPath("docker-compose.yaml"), "start")
			restartCmd.Stdout = os.Stdout
			restartCmd.Stderr = os.Stderr

			err = restartCmd.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}

	return exportCmd
}
