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

package validator

import (
	"embed"
	"html/template"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

var FilterValidation string

//go:embed *.tmpl
var validatorTemplate embed.FS

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "generates a custom report based on the results filter by this validation informed",
		Long: `use this command with the result of $audit index bundles [OPTIONS].
## When should I use this command?

If you are looking for to check all scenarios which fails with some specific warning or error message.
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
		"path of the JSON File result of the command audit-tool index bundles --index-image=<image> [OPTIONS]")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		log.Fatalf("Failed to mark `file` flag for `index` sub-command as required")
	}
	cmd.Flags().StringVar(&custom.Flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().StringVar(&custom.Flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().StringVar(&FilterValidation, "filter-validation", "",
		"filter by the error/warnings results which contain *filter-validation*")
	if err := cmd.MarkFlagRequired("filter-validation"); err != nil {
		log.Fatalf("Failed to mark `filter-validation` flag for `index` sub-command as required")
	}
	return cmd
}

func validation(cmd *cobra.Command, args []string) error {
	if len(custom.Flags.OutputPath) > 0 {
		if _, err := os.Stat(custom.Flags.OutputPath); os.IsNotExist(err) {
			return err
		}
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
	validatorReport := custom.NewValidatorReport(bundlesReport, custom.Flags.Filter, FilterValidation)

	log.Info("Generating output...")
	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(validatorReport.ImageName, "validator", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFS(validatorTemplate, "validator_template.go.tmpl"))
	err = t.Execute(f, validatorReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}
