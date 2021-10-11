// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Deprecated
// This script is helper to generate a txt file with all packages sorted by name
// which still without a compatible version with 4.9
// Example of usage: (see that we leave makefile target to help you out here)
// nolint: lll
// go run ./hack/scripts/packages/generate.go --image=testdata/reports/redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.9_2021-08-22.json
// go run ./hack/scripts/packages/generate.go --image=testdata/reports/redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.9_2021-08-22.json
// go run ./hack/scripts/packages/generate.go --image=testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8_2021-08-21.json
// go run ./hack/scripts/packages/generate.go --image=testdata/reports/redhat_community_operator_index/bundles_registry.redhat.io_redhat_community_operator_index_v4.8_2021-08-21.json
// todo: remove after 4.9-GA
package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/operator-framework/audit/hack"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
	log "github.com/sirupsen/logrus"
)

type File struct {
	APIDashReport                  *custom.APIDashReport
	MigrateNotIn49                 []custom.Migrated
	MigrateNotDefaultChannel       []custom.Migrated
	NotMigrateWithReplaces         []custom.NotMigrated
	NotMigrateWithReplacesAllHeads []custom.NotMigrated
	NotMigrateWithSkips            []custom.NotMigrated
	NotMigrateWithSkipsRange       []custom.NotMigrated
	NotMigrateWithSkipsAndRange    []custom.NotMigrated
	NotMigrateUnknow               []custom.NotMigrated
	TotalWorking49                 int
	NotMigratesMix                 []custom.NotMigrated
}

type ItemContact struct {
	Name     string
	Emails   []string
	Links    []string
	Warnings map[string]string
}

//nolint: lll,gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	defaultOutputPath := "hack/scripts/packages"

	var outputPath string
	var jsonFile string

	flag.StringVar(&outputPath, "output", defaultOutputPath, "Inform the path for output the report, if not informed it will be generated at hack/scripts/deprecated-bundles-repo/deprecate-green.")
	flag.StringVar(&jsonFile, "file", "", "Inform the path for the JSON result which will be used to generate the report. ")

	flag.Parse()

	byteValue, err := pkg.ReadFile(jsonFile)
	if err != nil {
		log.Fatal(err)
	}
	var bundlesReport bundles.Report

	err = json.Unmarshal(byteValue, &bundlesReport)
	if err != nil {
		log.Fatal(err)
	}

	apiDashReport, err := getAPIDashForImage(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	// Packages which has compatible version but none of them will end up on 4.9
	var migrateNotIn49 []custom.Migrated
	for _, v := range apiDashReport.Migrated {
		foundIn49 := false
		for _, b := range v.AllBundles {
			if len(b.KindsDeprecateAPIs) == 0 && (len(b.OCPLabel) == 0 || !pkg.IsOcpLabelRangeLowerThan49(b.OCPLabel)) {
				foundIn49 = true
				break
			}
		}
		if !foundIn49 {
			migrateNotIn49 = append(migrateNotIn49, v)
			continue
		}
	}

	// Packages which has compatible version but none of them will end up on 4.9
	var migrateNotDefaultChannel []custom.Migrated
	for _, v := range apiDashReport.Migrated {
		found := false
		for _, b := range v.AllBundles {
			if len(b.KindsDeprecateAPIs) == 0 && b.IsFromDefaultChannel {
				found = true
				break
			}
		}
		if !found {
			migrateNotDefaultChannel = append(migrateNotDefaultChannel, v)
			continue
		}
	}

	// Packages which does not nave any compatible version with 4.9 and are using replaces
	var notMigrateWithReplaces []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		foundReplace := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if len(b.Replace) > 0 {
				foundReplace = true
				break
			}
		}
		if foundReplace {
			notMigrateWithReplaces = append(notMigrateWithReplaces, v)
			continue
		}
	}

	var notMigrateWithSkips []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		foundSkips := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if len(b.SkipRange) > 0 && len(b.Replace) < 1 {
				foundSkips = true
				break
			}
		}
		if foundSkips {
			notMigrateWithSkips = append(notMigrateWithSkips, v)
			continue
		}
	}

	var notMigrateWithSkipRange []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		foundSkipRange := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if len(b.SkipRange) > 0 && len(b.Replace) < 1 {
				foundSkipRange = true
				break
			}
		}
		if foundSkipRange {
			notMigrateWithSkipRange = append(notMigrateWithSkipRange, v)
			continue
		}
	}

	var notMigrateWithSkipsAndRange []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		foundSkips := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if ((len(b.Skips) > 0) || len(b.SkipRange) > 0) && len(b.Replace) < 1 {
				foundSkips = true
				break
			}
		}
		if foundSkips {
			notMigrateWithSkipsAndRange = append(notMigrateWithSkipsAndRange, v)
			continue
		}
	}

	var notMigratesMix []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		found := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if len(b.Replace) > 0 && (len(b.Skips) > 0 || len(b.SkipRange) > 0) {
				found = true
				break
			}
		}
		if !found {
			notMigratesMix = append(notMigratesMix, v)
			continue
		}
	}

	var notMigrateUnknow []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		found := false
		for _, r := range notMigrateWithReplaces {
			if v.Name == r.Name {
				found = true
				break
			}
		}
		for _, s := range notMigrateWithSkipsAndRange {
			if v.Name == s.Name {
				found = true
				break
			}
		}
		if !found {
			notMigrateUnknow = append(notMigrateUnknow, v)
			continue
		}
	}

	var notMigrateWithReplacesAllHeads []custom.NotMigrated
	for _, v := range apiDashReport.NotMigrated {
		notFoundReplace := false
		headOfChannels := custom.GetHeadOfChannels(v.AllBundles)
		for _, b := range headOfChannels {
			if len(b.Replace) == 0 {
				notFoundReplace = true
				break
			}
		}
		if !notFoundReplace {
			notMigrateWithReplacesAllHeads = append(notMigrateWithReplacesAllHeads, v)
			continue
		}
	}

	sort.Slice(apiDashReport.NotMigrated[:], func(i, j int) bool {
		return apiDashReport.NotMigrated[i].Name < apiDashReport.NotMigrated[j].Name
	})
	sort.Slice(migrateNotIn49[:], func(i, j int) bool {
		return migrateNotIn49[i].Name < migrateNotIn49[j].Name
	})
	sort.Slice(migrateNotDefaultChannel[:], func(i, j int) bool {
		return migrateNotDefaultChannel[i].Name < migrateNotDefaultChannel[j].Name
	})
	sort.Slice(notMigrateWithReplaces[:], func(i, j int) bool {
		return notMigrateWithReplaces[i].Name < notMigrateWithReplaces[j].Name
	})
	sort.Slice(notMigrateWithReplacesAllHeads[:], func(i, j int) bool {
		return notMigrateWithReplacesAllHeads[i].Name < notMigrateWithReplacesAllHeads[j].Name
	})
	sort.Slice(notMigrateWithSkips[:], func(i, j int) bool {
		return notMigrateWithSkips[i].Name < notMigrateWithSkips[j].Name
	})
	sort.Slice(notMigrateWithSkipRange[:], func(i, j int) bool {
		return notMigrateWithSkipRange[i].Name < notMigrateWithSkipRange[j].Name
	})
	sort.Slice(notMigrateUnknow[:], func(i, j int) bool {
		return notMigrateUnknow[i].Name < notMigrateUnknow[j].Name
	})
	sort.Slice(notMigratesMix[:], func(i, j int) bool {
		return notMigratesMix[i].Name < notMigratesMix[j].Name
	})
	sort.Slice(notMigrateWithSkipsAndRange[:], func(i, j int) bool {
		return notMigrateWithSkipsAndRange[i].Name < notMigrateWithSkipsAndRange[j].Name
	})

	totalWorking49 := len(apiDashReport.Migrated) - len(migrateNotIn49)

	reportPath := filepath.Join(currentPath, hack.ReportsPath, "packages")
	command := exec.Command("mkdir", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	fp := filepath.Join(reportPath, pkg.GetReportName(apiDashReport.ImageName, "package", "txt"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/packages/template.go.tmpl")))
	err = t.Execute(f, File{APIDashReport: apiDashReport,
		MigrateNotIn49:                 migrateNotIn49,
		MigrateNotDefaultChannel:       migrateNotDefaultChannel,
		NotMigrateWithReplaces:         notMigrateWithReplaces,
		NotMigrateWithReplacesAllHeads: notMigrateWithReplacesAllHeads,
		TotalWorking49:                 totalWorking49,
		NotMigrateWithSkips:            notMigrateWithSkips,
		NotMigrateWithSkipsRange:       notMigrateWithSkipRange,
		NotMigrateWithSkipsAndRange:    notMigrateWithSkipsAndRange,
		NotMigrateUnknow:               notMigrateUnknow})
	if err != nil {
		panic(err)
	}

	// Generate the json files with contacts
	var all []ItemContact
	for _, v := range apiDashReport.NotMigrated {
		i := ItemContact{Name: v.Name}
		var emails []string
		var links []string
		warns := make(map[string]string, len(v.AllBundles))

		for _, b := range v.AllBundles {
			emails = append(emails, b.MaintainersEmail...)
			links = append(links, b.Links...)
			if len(b.BundleName) > 0 && b.ValidatorWarnings != nil {
				for _, w := range b.ValidatorWarnings {
					if strings.Contains(w, "1.22") {
						warns[b.BundleName] = strings.ReplaceAll(w, "this bundle", "this distribution")
					}
				}
			}
		}
		i.Emails = pkg.GetUniqueValues(emails)
		i.Links = pkg.GetUniqueValues(links)
		i.Warnings = warns

		all = append(all, i)
	}

	sort.Slice(all[:], func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	reportPath = filepath.Join(currentPath, hack.ReportsPath, "contacts")
	command = exec.Command("mkdir", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	fp = filepath.Join(reportPath, pkg.GetReportName(apiDashReport.ImageName, "contact", "json"))
	f, err = os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(all, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = hack.ReplaceInFile(fp, "", string(jsonResult))
	if err != nil {
		log.Fatal(err)
	}

}

func getAPIDashForImage(image string) (*custom.APIDashReport, error) {
	// Update here the path of the JSON report for the image that you would like to be used
	custom.Flags.File = image

	bundlesReport, err := custom.ParseBundlesJSONReport()
	if err != nil {
		log.Fatal(err)
	}

	apiDashReport := custom.NewAPIDashReport(bundlesReport)
	return apiDashReport, err
}
