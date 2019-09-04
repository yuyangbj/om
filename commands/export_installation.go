package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

//counterfeiter:generate -o ./fakes/export_installation_service.go --fake-name ExportInstallationService . exportInstallationService
type exportInstallationService interface {
	DownloadInstallationAssetCollection(outputFile string) error
}

func NewExportInstallation(service exportInstallationService, logger logger) *cobra.Command {
	var outputFile string
	//var omAPI exportInstallationService
	//var err error

	command := &cobra.Command{
		Use:     "export-installation",
		Short:   "exports the installation of the target Ops Manager",
		Long:    "This command will export the current installation of the target Ops Manager.",
		Example: "om version",
		//PreRunE: func(cmd *cobra.Command, args []string) error {
		//	omAPI, err = createAPI(cmd)
		//	return err
		//},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Printf("exporting installation")

			err := service.DownloadInstallationAssetCollection(outputFile)
			if err != nil {
				return fmt.Errorf("failed to export installation: %s", err)
			}

			logger.Printf("finished exporting installation")

			return nil
		},
	}

	command.Flags().StringVarP(&outputFile, "output-file", "o", "", "output path to write installation to")
	_ = command.MarkFlagRequired("output-file")

	return command
}
