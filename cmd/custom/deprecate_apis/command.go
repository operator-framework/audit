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
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

// BindFlags define the flags used to generate the bundle report
type BindFlags struct {
	File       string `json:"file"`
	OutputPath string `json:"outputPath"`
}

var flags = BindFlags{}

type Suggestion struct {
	Name            string
	Kinds           []string
	Bundles         []string
	BundlesMigrated []string
}

type OK struct {
	Name    string
	Kinds   []string
	Bundles []string
}

type Migrate struct {
	Name    string
	Kinds   []string
	Bundles []string
}

type APIDashReport struct {
	ImageName  string
	ImageID    string
	ImageHash  string
	ImageBuild string
	Migrate    []Migrate
	Suggestion []Suggestion
	OK         []OK
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deprecate-apis",
		Short: "generates a custom report based on defined criteria over the deprecated apis scnario for 1.22",
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

	cmd.Flags().StringVar(&flags.File, "file", "",
		"path of the JSON File result of the command audit-tool index bundles --index-image=<image> [OPTIONS]")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		log.Fatalf("Failed to mark `file` flag for `index` sub-command as required")
	}
	cmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	return cmd
}

func validation(cmd *cobra.Command, args []string) error {
	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting ...")
	byteValue, err := pkg.ReadFile(flags.File)
	if err != nil {
		log.Fatal(err)
	}
	var bundlesReport bundles.Report

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	err = json.Unmarshal(byteValue, &bundlesReport)
	if err != nil {
		log.Fatal(err)
	}
	depReport := buildReport(bundlesReport)

	reportFilePath := filepath.Join(flags.OutputPath,
		pkg.GetReportName(depReport.ImageName, "deprecate-apis", "html"))

	f, err := os.Create(reportFilePath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "/cmd/custom/deprecate_apis/template.go.tmpl")))
	err = t.Execute(f, depReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")

	return nil
}

//nolint:gocyclo
func buildReport(bundlesReport bundles.Report) APIDashReport {
	depReport := APIDashReport{}
	depReport.ImageName = bundlesReport.Flags.IndexImage
	depReport.ImageID = bundlesReport.IndexImageInspect.ID
	depReport.ImageBuild = bundlesReport.IndexImageInspect.DockerConfig.Labels["build-date"]

	// create a map with all bundles found per pkg name
	mapPackagesWithBundles := make(map[string][]bundles.Columns)
	for _, v := range bundlesReport.Columns {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}

	// all pkgs which has only deprecated APIs
	migratePkg := make(map[string][]bundles.Columns)
	for key, bundles := range mapPackagesWithBundles {
		foundMigrated := false
		foundSuggestion := false
		for _, b := range bundles {
			if hasAnySuggestion(b) {
				foundSuggestion = true
				continue
			}
			if len(b.KindsDeprecateAPIs) == 0 {
				foundMigrated = true
				continue
			}
		}
		if !foundMigrated && !foundSuggestion {
			migratePkg[key] = mapPackagesWithBundles[key]
		}
	}

	// all pkgs which has at leastt 1 bundle using deprecates and without apply
	// fully the suggestions and/or has not the head of channel migrated
	deprecatePkg := make(map[string][]bundles.Columns)
	for key, bundles := range mapPackagesWithBundles {
		found := false
		foundAnySuggestion := false
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 && b.IsDeprecationAPIsSuggestionsSet == pkg.GetYesOrNo(false) {
				found = true
			}
			if hasAnySuggestion(b) || len(b.KindsDeprecateAPIs) == 0 {
				foundAnySuggestion = true
			}
		}

		if found && foundAnySuggestion {
			deprecatePkg[key] = mapPackagesWithBundles[key]
		}
	}

	// all pkgs which has not any bundle with deprecate apis configured to be carry on to
	// 4.9
	okPkg := make(map[string][]bundles.Columns)
	for key, bundles := range mapPackagesWithBundles {
		isNotOK := false
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 && b.IsDeprecationAPIsSuggestionsSet == pkg.GetYesOrNo(false) {
				isNotOK = true
				break
			}
		}

		foundMigrated := false
		for _, b := range bundles {
			//some cases has not the pkg name we will skip that
			if len(b.PackageName) == 0 {
				continue
			}
			if len(b.KindsDeprecateAPIs) == 0 {
				foundMigrated = true
				break
			}
		}

		if !isNotOK && foundMigrated {
			okPkg[key] = mapPackagesWithBundles[key]
		}
	}

	for k, bundles := range okPkg {
		var kinds []string
		for _, b := range bundles {
			kinds = append(kinds, b.KindsDeprecateAPIs...)
		}
		var ok []string
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 {
				ok = append(ok, b.BundleName)
			}
		}
		depReport.OK = append(depReport.OK, OK{
			Name:    k,
			Kinds:   pkg.GetUniqueValues(kinds),
			Bundles: ok,
		})
	}

	for k, bundles := range migratePkg {
		var kinds []string
		for _, b := range bundles {
			kinds = append(kinds, b.KindsDeprecateAPIs...)
		}
		var migrate []string
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 {
				migrate = append(migrate, b.BundleName)
			}
		}

		sort.Slice(migrate[:], func(i, j int) bool {
			return migrate[i] < migrate[j]
		})

		depReport.Migrate = append(depReport.Migrate, Migrate{
			Name:    k,
			Kinds:   pkg.GetUniqueValues(kinds),
			Bundles: migrate,
		})
	}

	for k, bundles := range deprecatePkg {
		var kinds []string
		for _, b := range bundles {
			kinds = append(kinds, b.KindsDeprecateAPIs...)
		}
		var suggestion []string
		var migrated []string
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 {
				suggestion = append(suggestion,
					fmt.Sprintf("%s - (label:%s-max:%s)",
						b.BundleName,
						pkg.GetYesOrNo(pkg.IsOcpLabelRangeLowerThan49(b.OCPLabel)),
						pkg.GetYesOrNo(pkg.IsMaxOCPVersionLowerThan49(b.MaxOCPVersion))))
			} else {
				migrated = append(migrated, b.BundleName)
			}
		}

		sort.Slice(suggestion[:], func(i, j int) bool {
			return suggestion[i] < suggestion[j]
		})

		sort.Slice(migrated[:], func(i, j int) bool {
			return migrated[i] < migrated[j]
		})

		depReport.Suggestion = append(depReport.Suggestion, Suggestion{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Bundles:         suggestion,
			BundlesMigrated: migrated,
		})
	}

	return depReport
}

func hasAnySuggestion(b bundles.Columns) bool {
	if len(b.KindsDeprecateAPIs) > 0 {
		if pkg.IsOcpLabelRangeLowerThan49(b.OCPLabel) {
			return true
		}
		if pkg.IsMaxOCPVersionLowerThan49(b.MaxOCPVersion) {
			return true
		}
	}
	return false
}
