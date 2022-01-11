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

package deprecate

import (
	"fmt"
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
		Use: "deprecate-apis",
		Short: "generates a custom report based on defined criteria over the " +
			"deprecated apis scenario for 1.22 by default or the version informed",
		Long: "use this command with the result of `audit index bundles [OPTIONS]` " +
			"to check a dashboard in HTML format " +
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
	optionalValueEmpty := map[string]string{}
	cmd.Flags().StringToStringVarP(&custom.Flags.OptionalValues, "optional-values", "", optionalValueEmpty,
		"Inform a []string map of key=values which can be used by the report. e.g. to check the usage of deprecated APIs "+
			"against an Kubernetes version that it is intended to be distributed use `--optional-values=k8s-version=1.22`")
	cmd.Flags().StringVar(&custom.Flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
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

	apiDashReport := custom.NewAPIDashReport(bundlesReport, custom.Flags.OptionalValues, custom.Flags.Filter)

	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(apiDashReport.ImageName, fmt.Sprintf("deprecate-apis-%s", apiDashReport.K8SVersion), "html"))

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

//todo: this template requires to be embed
func getTemplatePath(currentPath string) string {
	return filepath.Join(currentPath, "/cmd/custom/deprecate/template.go.tmpl")
}
