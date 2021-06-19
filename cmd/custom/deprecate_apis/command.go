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

	"github.com/blang/semver"

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

type PartialComplying struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
}

type OK struct {
	Name            string
	Kinds           []string
	Bundles         []string
	Channels        []string
	BundlesMigrated []string
}

type NotComplying struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
}

type APIDashReport struct {
	ImageName        string
	ImageID          string
	ImageHash        string
	ImageBuild       string
	NotComplying     []NotComplying
	PartialComplying []PartialComplying
	GeneratedAt      string
	OK               []OK
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
	depReport.GeneratedAt = bundlesReport.GenerateAt

	// create a map with all bundlesPerPkg found per pkg name
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport.Columns {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}

	// (Red) Not complying
	// That are the packages which has none head channels compatible with 4.9 and/or configured accordingly
	// with max ocp version set
	notComplying := make(map[string][]bundles.Column)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		foundOK := 0
		foundConfiguredAccordingly := 0
		qtdHeads := 0
		for _, v := range bundlesPerPkg {
			if v.IsHeadOfChannel {
				qtdHeads++
			}
			foundOK, foundConfiguredAccordingly = getHeadOfChannelState(v, foundOK, foundConfiguredAccordingly)
		}
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if hasNotMigrated(bundlesPerPkg) {
				notComplying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		// If for the package has no bundle set in the channels
		// table as head of the channel then, we need to check
		// the scenarios
		if qtdHeads == 0 {
			// We need to check if the latest version for each
			// channel found is migrated or not
			bundlesPerChannels := make(map[string][]bundles.Column)
			for _, b := range bundlesPerPkg {
				for _, c := range b.Channels {
					bundlesPerChannels[c] = append(bundlesPerChannels[c], b)
				}
			}
			qtChannelOK := 0
			qtChannelConfiguredAccordingly := 0
			for _, bundlesFromChannel := range bundlesPerChannels {
				latest := getTheLatestBundleVersion(bundlesFromChannel)
				// check if latest is OK
				for _, v := range bundlesFromChannel {
					if v.BundleVersion == latest {
						// In this case has a valid path
						if len(v.KindsDeprecateAPIs) == 0 && !pkg.IsOcpLabelRangeLowerThan49(v.OCPLabel) {
							qtChannelOK++
						}
						// in this case will block the cluster upgrade at least
						if len(v.KindsDeprecateAPIs) > 0 && pkg.IsMaxOCPVersionLowerThan49(v.MaxOCPVersion) {
							qtChannelConfiguredAccordingly++
						}
						break
					}
				}
			}
			if qtChannelOK == 0 && qtChannelConfiguredAccordingly == 0 {
				notComplying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if foundOK == 0 && foundConfiguredAccordingly == 0 {
			notComplying[key] = mapPackagesWithBundles[key]
		}
	}

	// (Green) Complying
	// If is not using deprecated API(s) at all in the head channels
	// If has at least one channel head which is compatible with 4.9 (migrated)
	// and the other head channels are with max ocp version
	complying := make(map[string][]bundles.Column)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		foundOK := 0
		foundConfiguredAccordingly := 0
		qtdHeads := 0
		for _, v := range bundlesPerPkg {
			if v.IsHeadOfChannel {
				qtdHeads++
			}
			foundOK, foundConfiguredAccordingly = getHeadOfChannelState(v, foundOK, foundConfiguredAccordingly)
		}
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if !hasNotMigrated(bundlesPerPkg) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		// If for the package has no bundle set in the channels
		// table as head of the channel then, we need to check
		// the scenarios
		if qtdHeads == 0 {
			// We need to check if the latest version for each
			// channel found is migrated or not
			bundlesPerChannels := make(map[string][]bundles.Column)
			for _, b := range bundlesPerPkg {
				for _, c := range b.Channels {
					bundlesPerChannels[c] = append(bundlesPerChannels[c], b)
				}
			}
			qtChannelOK := 0
			qtChannelConfiguredAccordingly := 0
			for _, bundlesFromChannel := range bundlesPerChannels {
				latest := getTheLatestBundleVersion(bundlesFromChannel)
				// check if latest is OK
				for _, v := range bundlesFromChannel {
					if v.BundleVersion == latest {
						// In this case has a valid path
						if len(v.KindsDeprecateAPIs) == 0 && !pkg.IsOcpLabelRangeLowerThan49(v.OCPLabel) {
							qtChannelOK++
						}
						// in this case will block the cluster upgrade at least
						if len(v.KindsDeprecateAPIs) > 0 && pkg.IsMaxOCPVersionLowerThan49(v.MaxOCPVersion) {
							qtChannelConfiguredAccordingly++
						}
						break
					}
				}
			}

			if len(bundlesPerChannels) == qtChannelOK ||
				(qtChannelOK > 0 && len(bundlesPerChannels) == qtChannelOK+qtChannelConfiguredAccordingly) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if qtdHeads == foundOK || (foundOK > 0 && qtdHeads == foundOK+foundConfiguredAccordingly) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}

	// (Amber) Partial complying
	// if is not read or green then fail in the amber scenarios
	partialComplying := make(map[string][]bundles.Column)
	for key := range mapPackagesWithBundles {
		if !(len(complying[key]) > 0 || len(notComplying[key]) > 0) {
			partialComplying[key] = mapPackagesWithBundles[key]
		}
	}

	for k, bundles := range complying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		depReport.OK = append(depReport.OK, OK{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}

	for k, bundles := range notComplying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		depReport.NotComplying = append(depReport.NotComplying, NotComplying{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}

	for k, bundles := range partialComplying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		depReport.PartialComplying = append(depReport.PartialComplying, PartialComplying{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}

	return depReport
}

func getTheLatestBundleVersion(bundlesFromChannel []bundles.Column) string {
	var latestVersion string
	for _, v := range bundlesFromChannel {
		bundleVersionSemVer, _ := semver.ParseTolerant(v.BundleVersion)
		latestVersionSemVer, _ := semver.ParseTolerant(latestVersion)
		if bundleVersionSemVer.GT(latestVersionSemVer) {
			latestVersion = v.BundleVersion
		}
	}
	return latestVersion
}

func hasNotMigrated(bundlesPerPkg []bundles.Column) bool {
	foundNotMigrated := false
	for _, v := range bundlesPerPkg {
		if len(v.KindsDeprecateAPIs) > 0 {
			foundNotMigrated = true
			break
		}
	}
	return foundNotMigrated
}

func getHeadOfChannelState(v bundles.Column, foundOK int, foundConfiguredAccordingly int) (int, int) {
	if v.IsHeadOfChannel {
		// In this case has a valid path
		if len(v.KindsDeprecateAPIs) == 0 && !pkg.IsOcpLabelRangeLowerThan49(v.OCPLabel) {
			foundOK++
		}
		// in this case will block the cluster upgrade at least
		if len(v.KindsDeprecateAPIs) > 0 && pkg.IsMaxOCPVersionLowerThan49(v.MaxOCPVersion) {
			foundConfiguredAccordingly++
		}
	}
	return foundOK, foundConfiguredAccordingly
}

func getReportValues(bundles []bundles.Column) ([]string, []string, []string, []string) {
	var kinds []string
	var channels []string
	for _, b := range bundles {
		kinds = append(kinds, b.KindsDeprecateAPIs...)
	}
	for _, b := range bundles {
		channels = append(channels, b.Channels...)
	}
	var bundlesNotMigrated []string
	var bundlesMigrated []string
	for _, b := range bundles {
		if len(b.KindsDeprecateAPIs) > 0 {
			bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b))
		} else {
			bundlesMigrated = append(bundlesMigrated, buildBundleString(b))
		}
	}

	sort.Slice(bundlesNotMigrated[:], func(i, j int) bool {
		return bundlesNotMigrated[i] < bundlesNotMigrated[j]
	})

	sort.Slice(bundlesMigrated[:], func(i, j int) bool {
		return bundlesMigrated[i] < bundlesMigrated[j]
	})

	return kinds, channels, bundlesNotMigrated, bundlesMigrated
}

func getMaxOCPValue(b bundles.Column) string {
	maxValue := b.MaxOCPVersion
	if len(maxValue) == 0 {
		maxValue = "not set"
	}
	return maxValue
}

func buildBundleString(b bundles.Column) string {
	return fmt.Sprintf("%s - (label=%s,max=%s,channels=%s,head:%s)",
		b.BundleName,
		b.OCPLabel,
		getMaxOCPValue(b),
		pkg.GetUniqueValues(b.Channels),
		pkg.GetYesOrNo(b.IsHeadOfChannel),
	)
}
