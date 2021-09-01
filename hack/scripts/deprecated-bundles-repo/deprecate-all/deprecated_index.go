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
// This script is only a helper for we are able to know what are the bundles that we need to
// deprecated on 4.9. That will be removed as soon as possible and is just added
// here in case it be required to be checked and used so far.
// The following script uses the JSON format output image to
// generates the yml file with all bundles which requires
// to be deprecated because are using the APIs which will be removed on ocp 4.9 .
// Example of usage: (see that we leave makefile target to help you out here)
// nolint: lll
// go run ./hack/scripts/deprecated-bundles-repo/deprecate-all/deprecated_index.go --image=testdata/reports/redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.8_2021-08-10.json
// go run ./hack/scripts/deprecated-bundles-repo/deprecate-all/deprecated_index.go --image=testdata/reports/redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8_2021-08-06.json
// go run ./hack/scripts/deprecated-bundles-repo/deprecate-all/deprecated_index.go --image=testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8_2021-08-15.json
// todo: remove after 4.9-GA
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/operator-framework/audit/hack"

	"github.com/operator-framework/audit/pkg/reports/custom"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Bundles struct {
	Details string
	Paths   string
}

type Deprecated struct {
	PackageName string
	Bundles     []Bundles
}

type File struct {
	Deprecated    []Deprecated
	APIDashReport *custom.APIDashReport
}

//nolint: lll,gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	defaultOutputPath := "hack/scripts/deprecated-bundles-repo/deprecate-all"

	var outputPath string
	var jsonFile string

	flag.StringVar(&outputPath, "output", defaultOutputPath, "Inform the path for output the report, if not informed it will be generated at hack/scripts/deprecated-bundles-repo/deprecate-green.")
	flag.StringVar(&jsonFile, "image", "", "Inform the path for the JSON result which will be used to generate the report. ")

	flag.Parse()

	byteValue, err := pkg.ReadFile(filepath.Join(currentPath, jsonFile))
	if err != nil {
		log.Fatal(err)
	}
	var bundlesReport bundles.Report

	err = json.Unmarshal(byteValue, &bundlesReport)
	if err != nil {
		log.Fatal(err)
	}

	// create a map with all bundles found per pkg name
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport.Columns {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}

	// some pkgs name are empty, the following code checks what is the package by looking
	// into the bundle path and fixes that
	for _, bundle := range mapPackagesWithBundles[""] {
		split := strings.Split(bundle.BundleImagePath, "/")
		nm := ""
		for _, v := range split {
			if strings.Contains(v, "@") {
				nm = strings.Split(bundle.BundleImagePath, "@")[0]
				break
			}
		}
		for key, bundles := range mapPackagesWithBundles {
			for _, b := range bundles {
				if strings.Contains(b.BundleImagePath, nm) {
					mapPackagesWithBundles[key] = append(mapPackagesWithBundles[key], bundle)
				}
			}
		}

		//remove from the empty key
		var all []bundles.Column
		for _, be := range mapPackagesWithBundles[""] {
			if be.BundleImagePath != bundle.BundleImagePath {
				all = append(all, be)
			}
		}
		mapPackagesWithBundles[""] = all
	}

	apiDashReport, err := getAPIDashForImage(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	// filter by all pkgs which has only deprecated APIs
	hasDeprecated := make(map[string][]bundles.Column)
	for key, bundles := range mapPackagesWithBundles {
		for _, b := range bundles {
			// If has UNKNOW status we need ignore that means an error when we tried to check
			// the results
			if len(b.KindsDeprecateAPIs) > 0 && len(b.DeprecateAPIsManifests[pkg.Unknown]) == 0 {
				hasDeprecated[key] = mapPackagesWithBundles[key]
			}
		}
	}

	// create the object with the bundle path
	// see that we need to remove the redhat registry domain
	allDeprecated := []Deprecated{}
	for key, bundles := range hasDeprecated {

		// do not add a package that has no bundles to be deprecated
		found := false
		for _, b := range bundles {
			// We just ONLY the bundles which are using the removed APIS
			if len(b.KindsDeprecateAPIs) > 0 && len(b.DeprecateAPIsManifests[pkg.Unknown]) == 0 {
				found = true
			}
		}
		if !found {
			continue
		}

		deprecatedYaml := Deprecated{PackageName: key}

		// nolint:scopelint
		sort.Slice(bundles[:], func(i, j int) bool {
			return bundles[i].BundleName < bundles[j].BundleName
		})

		for _, b := range bundles {
			// We just ONLY the bundles which are using the removed APIS
			if len(b.KindsDeprecateAPIs) > 0 && len(b.DeprecateAPIsManifests[pkg.Unknown]) == 0 {
				deprecatedYaml.Bundles = append(deprecatedYaml.Bundles,
					Bundles{
						Paths:   b.BundleImagePath,
						Details: b.BundleName,
					})
			}
		}
		allDeprecated = append(allDeprecated, deprecatedYaml)
	}

	sort.Slice(allDeprecated[:], func(i, j int) bool {
		return allDeprecated[i].PackageName < allDeprecated[j].PackageName
	})

	reportPath := filepath.Join(currentPath, hack.ReportsPath, "deprecate-all")
	command := exec.Command("mkdir", reportPath)
	_, _ = pkg.RunCommand(command)

	fp := filepath.Join(reportPath, pkg.GetReportName(apiDashReport.ImageName, "deprecated", "yml"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/deprecated-bundles-repo/deprecate-all/template.go.tmpl")))
	err = t.Execute(f, File{Deprecated: allDeprecated, APIDashReport: apiDashReport})
	if err != nil {
		panic(err)
	}

	//To generate in the JSON format to allow us to do the test
	var onlyBundles []string
	for _, a := range allDeprecated {
		for _, ab := range a.Bundles {
			onlyBundles = append(onlyBundles, ab.Paths)
		}

	}

	reportPath = filepath.Join(currentPath, hack.ReportsPath, "deprecate-json")
	command = exec.Command("mkdir", reportPath)
	_, _ = pkg.RunCommand(command)

	fp = filepath.Join(reportPath, pkg.GetReportName(apiDashReport.ImageName, "deprecate-all", "json"))
	f, err = os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(onlyBundles, "", "\t")
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
