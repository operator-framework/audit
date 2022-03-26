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
	"embed"
	"fmt"
	"os"

	"html/template"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

//go:embed *.tmpl
var deprecateTemplate embed.FS

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deprecate-apis",
		Short: "generates a custom report to check packages impact by k8s apis removal.",
		Long: `use this command with the result of $audit index bundles [OPTIONS].

## When should I use this command?

if you are looking for check what are the packages and head of channels which uses the api removed 
in the k8s 1.22 

OR 

which requests permissions in the API which will be removed in the k8s version 1.25 and 1.26.

**IMPORTANT** Note that for the k8s versions 1.25 and 1.26 we are unable to check what bundles 
can or not work on these release. It is very unlike author add manifests with the APIs affected 
and in many cases the Kinds are indeed not supported. 

## How the check is done for 1.25 and 1.26?

For these versions audit tool will check the cases where the bundles are asking permissions 
to the APIs affected by looking at the rules (RBCA). However, by looking at the rules we 
are unable to know if the Operator requires the versions which will be removed or not since
the version is not present. 

**For example:** The RBAC to create, patch, delete a CronJob can be checked but it does not
have the versions so that, we are unable to know if the Operator still using v1beta1 
which will be removed or v1 which will work on these versions. 

However, it allows us to check what are the packages which are more likely to be impacted.

## How to inform the version? 

Use the --optional-values flag and the key k8s-version to inform the version which should 
be used to generate the report, see: 

- For 1.22 : --optional-values=k8s-version=1.22
- For 1.25 : --optional-values=k8s-version=1.25
- For 1.26 : --optional-values=k8s-version=1.26
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

	t := template.Must(template.ParseFS(deprecateTemplate, "deprecate_template.go.tmpl"))
	err = t.Execute(f, apiDashReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}
