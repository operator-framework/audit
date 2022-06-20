// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this File except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package multiarch

import (
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//go:embed *.tmpl
var multiarchTemplate embed.FS

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multiarch",
		Short: "generates a custom report based on defined criteria over Multiple Architectures",
		Long: `use this command with the result of $audit index bundles [OPTIONS].
## When should I use this command?

If you are looking for:
- verify what are the packages which has head of channels that are not providing
support for Multiple Architectures.
- verify what are the packages which has head of channels that probably are not
providing a valid configuration for this criteria
`,
		PreRunE: validation,
		RunE:    run,
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	cmd.Flags().StringVar(&custom.Flags.File, "file", "",
		"path of the JSON File result of the command audit-tool index bundles --index-image=<image>"+
			" [OPTIONS]")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		log.Fatalf("Failed to mark `file` flag for `index` sub-command as required")
	}
	cmd.Flags().StringVar(&custom.Flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().StringVar(&custom.Flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().StringVar(&custom.Flags.ContainerEngine, "container-engine", pkg.Docker,
		fmt.Sprintf("specifies the container tool to use. If not set, the default value is docker. "+
			"Note that you can use the environment variable CONTAINER_ENGINE to inform this option. "+
			"[Options: %s and %s]", pkg.Docker, pkg.Podman))

	return cmd
}

func validation(cmd *cobra.Command, args []string) error {
	if len(custom.Flags.OutputPath) > 0 {
		if _, err := os.Stat(custom.Flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	if len(custom.Flags.ContainerEngine) == 0 {
		custom.Flags.ContainerEngine = pkg.GetContainerToolFromEnvVar()
	}
	if custom.Flags.ContainerEngine != pkg.Docker && custom.Flags.ContainerEngine != pkg.Podman {
		return fmt.Errorf("invalid value for the flag --container-engine (%s)."+
			" The valid options are %s and %s", custom.Flags.ContainerEngine, pkg.Docker, pkg.Podman)
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting ...")

	bundlesReport, err := custom.ParseBundlesJSONReport()
	if err != nil {
		return err
	}

	log.Info("Generating data...")

	multiarchReport := custom.NewMultipleArchitecturesReport(bundlesReport, custom.Flags.Filter,
		custom.Flags.ContainerEngine)

	log.Info("Generating output...")
	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(multiarchReport.ImageName, "multiarch", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFS(multiarchTemplate, "multiarch_template.go.tmpl"))
	err = t.Execute(f, multiarchReport)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}
