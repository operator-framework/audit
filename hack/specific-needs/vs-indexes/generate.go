// Copyright 2022 The Audit Authors
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

// It is a helper script to compare the RedHat vs Community catalog and find
// the content which is from RedHat and exist in both indexes.

// The following script helps us to check solutions that were published
// in the index 4.N but are not in the 4.N+1. It is specific for the scenarios
// checked with this tool however, that seems a valid feature to be added
// as subcommand.

// TODO: Move this check for a subcommand where it would be able to check
// and return the result of the scenarios which are in the indexA informed
// but cannot be found in the indexB.
package main

import (
	"fmt"
	semverv4 "github.com/blang/semver/v4"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	//"sort"
	"strings"
	"time"

	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

//nolint:gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath)
	catalogsPath := filepath.Join(fullReportsPath, "vs-index")

	command := exec.Command("rm", "-rf", catalogsPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", catalogsPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	custom.Flags.OutputPath = catalogsPath

	// certified

	err = generateReportFor("certified",
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.9.json"),
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.10.json"))
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("certified",
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.8.json"),
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.9.json"))
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("certified",
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.7.json"),
		filepath.Join(fullReportsPath,"redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.8.json"))
	if err != nil {
		log.Error(err)
	}

	// marketplace

	err = generateReportFor("marketplace",
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.7.json"),
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8.json"))
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("marketplace",
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8.json"),
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.9.json"))
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("marketplace",
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.9.json"),
		filepath.Join(fullReportsPath,"redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8.json"))
	if err != nil {
		log.Error(err)
	}

	// redhat

	err = generateReportFor("redhat",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.7.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8.json")
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("redhat",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.9.json")
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("redhat",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.9.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.10.json")
	if err != nil {
		log.Error(err)
	}

	// community

	err = generateReportFor("community",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.7.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.8.json")
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("community",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.8.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.9.json")
	if err != nil {
		log.Error(err)
	}

	err = generateReportFor("community",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.9.json",
		"/Users/camilamacedo86/go/src/github.com/operator-framework/audit/testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.10.json")
	if err != nil {
		log.Error(err)
	}
}

func generateReportFor(subdir, pathLowVersion, pathNextVersion string) error {

	custom.Flags.File = pathLowVersion
	allBundlesReportForLowerVersion, err := custom.ParseBundlesJSONReport()
	if err != nil {
		return err
	}

	custom.Flags.File = pathNextVersion
	allBundlesReportForNextVersion, err := custom.ParseBundlesJSONReport()
	if err != nil {
		return err
	}

	vsIndexReport := newVsIndexReport(allBundlesReportForLowerVersion, allBundlesReportForNextVersion)

	tagLower:= strings.Split(allBundlesReportForLowerVersion.Flags.IndexImage, "v")[1]
	tagNext:= strings.Split(allBundlesReportForNextVersion.Flags.IndexImage, "v")[1]
	name := fmt.Sprintf("%s_from_%s_to_%s", subdir, tagLower, tagNext)
	reportName := strings.ReplaceAll(name, " ", "_")


	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath)
	catalogsPath := filepath.Join(fullReportsPath, "vs-index", subdir)

	command := exec.Command("mkdir", catalogsPath)
	_, err = pkg.RunCommand(command)
	custom.Flags.OutputPath = catalogsPath

	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(reportName, "vs_index", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(getTemplatePath()))
	err = t.Execute(f, vsIndexReport)
	if err != nil {
		panic(err)
	}

	f.Close()
	log.Infof("Operation completed.")
	return nil
}

func getTemplatePath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(currentPath, "/hack/specific-needs/vs-indexes/template.go.tmpl")
}

type RedHadPackages struct {
	PackageName     string
	BundlesWithOCPLabel []string
}

type VSReport struct {
	RedHadPackages []RedHadPackages
	GeneratedAt    string
	ImageNames    []string
}

// nolint:gocyclo
func newVsIndexReport(bundlesReportLowerVersion, bundlesReportNextVersion bundles.Report) *VSReport {
	vsReport := VSReport{}
	dt := time.Now().Format("2006-01-02")
	vsReport.GeneratedAt = dt
	vsReport.ImageNames = []string{bundlesReportLowerVersion.Flags.IndexImage,bundlesReportNextVersion.Flags.IndexImage}

	pkgNotFound := []RedHadPackages{}
	mapPackagesWithBundlesLowerVersion := MapBundlesPerPackage(bundlesReportLowerVersion.Columns)
	for pkg, entries := range mapPackagesWithBundlesLowerVersion {
		headOfChannels := GetHeadOfChannels(entries)
		mapPackagesWithBundlesLowerVersion[pkg] = headOfChannels
	}

	mapPackagesWithBundlesNextVersion := MapBundlesPerPackage(bundlesReportNextVersion.Columns)
	for pkg, entries := range mapPackagesWithBundlesNextVersion {
		headOfChannels := GetHeadOfChannels(entries)
		mapPackagesWithBundlesNextVersion[pkg] = headOfChannels
	}

	for k, v := range mapPackagesWithBundlesLowerVersion {
		if k == ""{
			continue
		}
		if len(mapPackagesWithBundlesNextVersion[k]) == 0 {
			pkg := RedHadPackages{}
			pkg.PackageName = k
			res := []string{}
			for _, bundles := range v {
				res = append(res, buildBundleString(bundles))
			}
			pkg.BundlesWithOCPLabel = res
			pkgNotFound = append(pkgNotFound, pkg)
		}
	}

	vsReport.RedHadPackages = pkgNotFound

	return &vsReport
}

func buildBundleString(b bundles.Column) string {
	const OCPLabel = "com.redhat.openshift.versions"

	return fmt.Sprintf("%s - (ocp label=%s - max=%s)",
		b.BundleCSV.Name,
		b.BundleImageLabels[OCPLabel],
		GetMaxOCPValue(b),
	)
}


// GetMaxOCPValue returns the Max OCP annotation find on the bundle or an string not set to define
// that it was not set
func GetMaxOCPValue(b bundles.Column) string {
	maxValue := b.MaxOCPVersion
	if len(maxValue) == 0 {
		maxValue = "not set"
	}
	return maxValue
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func MapBundlesPerPackage(bundlesReport []bundles.Column) map[string][]bundles.Column {
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}
	return mapPackagesWithBundles
}

func GetHeadOfChannels(bundlesOfPackage []bundles.Column) []bundles.Column {
	var headOfChannels []bundles.Column
	qtdHeads := 0
	for _, v := range bundlesOfPackage {
		if v.IsHeadOfChannel {
			qtdHeads++
			headOfChannels = append(headOfChannels, v)
		}
	}

	bundlesPerChannels := BuildMapBundlesPerChannels(bundlesOfPackage)

	// If for the package has no bundle set in the channels
	// table as head of the channel then, we need to check
	// the scenarios
	if qtdHeads == 0 || qtdHeads != len(bundlesPerChannels) {
		headOfChannels = GetLatestBundlesVersions(bundlesPerChannels)
	}
	return headOfChannels
}

// BuildMapBundlesPerChannels returns a map of bundles per packages
func BuildMapBundlesPerChannels(bundlesPerPkg []bundles.Column) map[string][]bundles.Column {
	bundlesPerChannels := make(map[string][]bundles.Column)
	for _, b := range bundlesPerPkg {
		for _, c := range b.Channels {
			bundlesPerChannels[c] = append(bundlesPerChannels[c], b)
		}
	}
	return bundlesPerChannels
}


// GetQtLatestVersionChannelsState returns the qtd. of channels which are OK and configured with max ocp version
func GetLatestBundlesVersions(bundlesPerChannels map[string][]bundles.Column) []bundles.Column {
	var latestBundlesVersionsPerChannel []bundles.Column
	for _, bundlesFromChannel := range bundlesPerChannels {
		latest := GetTheLatestBundleVersion(bundlesFromChannel)
		for _, bd := range bundlesFromChannel {

			if bd.BundleCSV == nil {
				continue
			}

			if bd.BundleCSV.Spec.Version.String() == latest {
				latestBundlesVersionsPerChannel = append(latestBundlesVersionsPerChannel, bd)
				continue
			}
		}
	}
	return latestBundlesVersionsPerChannel
}


// GetTheLatestBundleVersion returns the latest/upper semversion
func GetTheLatestBundleVersion(bundlesFromChannel []bundles.Column) string {
	latestVersion, _ := semverv4.ParseTolerant("0.0.0")
	for _, v := range bundlesFromChannel {

		if v.BundleCSV == nil {
			continue
		}

		if v.BundleCSV.Spec.Version.Version.GT(latestVersion) {
			latestVersion = v.BundleCSV.Spec.Version.Version
		}
	}
	return latestVersion.String()
}
