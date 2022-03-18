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

package catalogs

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/audit/pkg/reports/alpha"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

var name string

// TODO: move this logic to the hack /hack/catalogs/generate.go since it is a helper for
// an specific need and not a generic report OR make it generic enough
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "catalogs",
		Short:   "[Do not use it] - generates the RedHat vs Community report",
		PreRunE: validation,
		RunE:    run,
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	cmd.Flags().StringVar(&custom.Flags.Files, "files", "",
		"path of the JSON File(s) using result of the command audit-tool "+
			"index bundles --index-image=<image> [OPTIONS]")
	if err := cmd.MarkFlagRequired("files"); err != nil {
		log.Fatalf("Failed to mark `file` flag for `index` sub-command as required")
	}
	cmd.Flags().StringVar(&custom.Flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().StringVar(&custom.Flags.Template, "template", "",
		"inform the path of the template that should be used. If not informed the default will be used")
	cmd.Flags().StringVar(&custom.Flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().StringVar(&name, "name", "",
		"inform the report name")
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

	allBundlesReport, err := custom.ParseMultiBundlesJSONReport()
	if err != nil {
		return err
	}

	catalogReport := alpha.NewCatalogReportReport(allBundlesReport, custom.Flags.Filter, name)

	reportName := strings.ReplaceAll(name, " ", "_")
	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(reportName, "catalog", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	if len(custom.Flags.Template) == 0 {
		custom.Flags.Template = getTemplatePath(currentPath)
	}

	t := template.Must(template.ParseFiles(custom.Flags.Template))
	err = t.Execute(f, catalogReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}

// todo: this template requires to be embed
func getTemplatePath(currentPath string) string {
	return filepath.Join(currentPath, "/cmd/alpha/catalogs/template.go.tmpl")
}
