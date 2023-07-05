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
			nemoContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_nemonode_1")
			nemoContainer, err := nemoContainerIDCmd.Output()
			if err != nil {
				return err
			}

			ibcChainContainerIDCmd := exec.Command("docker", "ps", "-aqf", "name=generated_ibcnode_1")
			ibcContainer, err := ibcChainContainerIDCmd.Output()
			if err != nil {
				return err
			}

			makeNewNemoImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(nemoContainer)), "nemo-export-temp")

			nemoImageOutput, err := makeNewNemoImageCmd.Output()
			if err != nil {
				return err
			}

			makeNewIbcImageCmd := exec.Command("docker", "commit", strings.TrimSpace(string(ibcContainer)), "ibc-export-temp")
			ibcImageOutput, err := makeNewIbcImageCmd.Output()
			if err != nil {
				return err
			}

			localNemoMountPath := generatedPath("nemo", "initstate", ".nemo", "config")
			localIbcMountPath := generatedPath("ibcchain", "initstate", ".nemo", "config")

			nemoExportCmd := exec.Command(
				"docker", "run",
				"-v", strings.TrimSpace(fmt.Sprintf("%s:/root/.nemo/config", localNemoMountPath)),
				"nemo-export-temp",
				"nemo", "export")
			nemoExportJSON, err := nemoExportCmd.Output()
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
			nemoFilename := fmt.Sprintf("nemo-export-%d.json", ts)
			ibcFilename := fmt.Sprintf("ibc-export-%d.json", ts)

			fmt.Printf("Created exports %s and %s\nCleaning up...", nemoFilename, ibcFilename)

			err = os.WriteFile(nemoFilename, nemoExportJSON, 0644)
			if err != nil {
				return err
			}
			err = os.WriteFile(ibcFilename, ibcExportJSON, 0644)
			if err != nil {
				return err
			}

			// docker ps -aqf "name=containername"
			tempNemoContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=nemo-export-temp")
			tempNemoContainer, err := tempNemoContainerIDCmd.Output()
			if err != nil {
				return err
			}
			tempIbcContainerIDCmd := exec.Command("docker", "ps", "-aqf", "ancestor=ibc-export-temp")
			tempIbcContainer, err := tempIbcContainerIDCmd.Output()
			if err != nil {
				return err
			}

			deleteNemoContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempNemoContainer)))
			err = deleteNemoContainerCmd.Run()
			if err != nil {
				return err
			}
			deleteIbcContainerCmd := exec.Command("docker", "rm", strings.TrimSpace(string(tempIbcContainer)))
			err = deleteIbcContainerCmd.Run()
			if err != nil {
				return err
			}

			deleteNemoImageCmd := exec.Command("docker", "rmi", strings.TrimSpace(string(nemoImageOutput)))
			err = deleteNemoImageCmd.Run()
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
