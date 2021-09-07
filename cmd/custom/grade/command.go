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

package grade

import (
	"html/template"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "grade",
		Short: "it is an experimental custom dashboard which generates a custom report based on defined " +
			"criteria over the grade quality of the packages",
		Long: "use this command with the result of `audit index bundles [OPTIONS]` to check a dashboard in HTML format " +
			"with the packages data",
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
	cmd.Flags().StringVar(&custom.Flags.Template, "template", "",
		"inform the path of the template that should be used. If not informed the default will be used")

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

	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}

	bundlesReport, err := custom.ParseBundlesJSONReport()
	if err != nil {
		return err
	}

	apiDashReport := custom.NewGradeReport(bundlesReport)

	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(apiDashReport.ImageName, "grade", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	if len(custom.Flags.Template) == 0 {
		custom.Flags.Template = getTemplatePath(currentPath)
	}

	t := template.Must(template.ParseFiles(custom.Flags.Template))
	err = t.Execute(f, apiDashReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}

func getTemplatePath(currentPath string) string {
	return filepath.Join(currentPath, "/cmd/custom/grade/template.go.tmpl")
}
