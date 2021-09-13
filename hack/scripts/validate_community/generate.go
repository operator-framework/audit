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
// This script is only a helper for we check the community index
// todo: remove after 4.9-GA

package main

import (
	"flag"
	"fmt"
	"html/template"

	log "github.com/sirupsen/logrus"

	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type Result struct {
	PackageName string
	Bundles     []bundles.Column
}

type File struct {
	ShouldExist    []Result
	ShouldNotExist []Result
}

//nolint:lll,gocyclo,govet
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var image string
	var outputPath string

	flag.StringVar(&outputPath, "output", currentPath, "Inform the path for output the report, if not informed it will be generated at hack/scripts/deprecated-bundles-repo/deprecate-green.")
	flag.StringVar(&image, "image", "", "inform the final image result")

	flag.Parse()

	binPath := filepath.Join(currentPath, hack.BinPath)
	command := exec.Command(binPath, "index", "bundles",
		fmt.Sprintf("--index-image=%s", "registry.redhat.io/redhat/community-operator-index:v4.8"),
		"--output=json",
		"--disable-scorecard",
		fmt.Sprintf("--output-path=%s", outputPath),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command(binPath, "index", "bundles",
		fmt.Sprintf("--index-image=%s", image),
		"--output=json",
		"--disable-scorecard",
		fmt.Sprintf("--output-path=%s", outputPath),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	name48report := pkg.GetReportName("registry.redhat.io/redhat/community-operator-index:v4.8", "bundles", "json")
	apiDashReport48, err := getAPIDashForImage(filepath.Join(outputPath, name48report))
	if err != nil {
		log.Fatal(err)
	}

	nameFinalJSONReport := pkg.GetReportName(image, "bundles", "json")
	apiDashReportFinalResult, err := getAPIDashForImage(filepath.Join(outputPath, nameFinalJSONReport))
	if err != nil {
		log.Fatal(err)
	}

	command = exec.Command(binPath, "dashboard", "deprecate-apis",
		fmt.Sprintf("--file=%s", nameFinalJSONReport),
		fmt.Sprintf("--output-path=%s", outputPath),
	)

	log.Infof("You can check the HTML report for the final result in %", filepath.Join(outputPath, pkg.GetReportName(image, "deprecate-apis", "html")))

	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	var shouldNoExist []bundles.Column

	// all that is not migrated and is not deprecated should not exist
	for _, k := range apiDashReportFinalResult.NotMigrated {
		for _, v := range k.AllBundles {
			if !v.IsDeprecated && len(v.PackageName) > 0 {
				shouldNoExist = append(shouldNoExist, v)
			}
		}
	}

	// all bundle that is from the migrated package
	// and uses the deprecate apis and is not migrated should not exist
	// also, all that has OCP label set < 4.9 should not exist
	for _, m := range apiDashReport48.Migrated {
		for _, mb := range m.AllBundles {
			if !mb.IsDeprecated && len(mb.PackageName) > 0 && len(mb.DeprecateAPIsManifests) > 0 {
				for _, f := range apiDashReportFinalResult.Migrated {
					for _, fb := range f.AllBundles {
						if fb.BundleName == mb.BundleName && len(fb.PackageName) > 0 && !fb.IsDeprecated {
							ocpLabel := fb.OCPLabel
							if len(ocpLabel) == 0 {
								ocpLabel = fb.OCPLabelAnnotations
							}
							if len(fb.DeprecateAPIsManifests) > 0 || !pkg.IsOcpLabelRangeLowerThan49(ocpLabel) {
								shouldNoExist = append(shouldNoExist, fb)
							}
						}
					}
				}
			}
		}
	}

	// Let's check what should exist
	var shouldExist []bundles.Column
	for _, m := range apiDashReport48.Migrated {
		for _, mb := range m.AllBundles {
			ocpLabel := mb.OCPLabel
			if len(ocpLabel) == 0 {
				ocpLabel = mb.OCPLabelAnnotations
			}
			if !mb.IsDeprecated && len(mb.PackageName) > 0 && len(mb.DeprecateAPIsManifests) == 0 && !pkg.IsOcpLabelRangeLowerThan49(ocpLabel) {
				for _, f := range apiDashReportFinalResult.Migrated {
					found := false
					for _, fb := range f.AllBundles {
						if fb.BundleName == mb.BundleName {
							found = true
							break
						}
						if !found {
							shouldExist = append(shouldExist, fb)
						}
					}
				}
			}
		}
	}

	fp := filepath.Join(outputPath, pkg.GetReportName(image, "validate", "yml"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	// create a map with all bundles found per pkg name
	mapPackagesWithBundlesShouldNoExist := make(map[string][]bundles.Column)
	for _, v := range shouldNoExist {
		mapPackagesWithBundlesShouldNoExist[v.PackageName] = append(mapPackagesWithBundlesShouldNoExist[v.PackageName], v)
	}

	shouldNotExistResult := []Result{}
	for key, bundles := range mapPackagesWithBundlesShouldNoExist {
		shouldNotExistResult = append(shouldNotExistResult, Result{PackageName: key, Bundles: bundles})
	}

	// create a map with all bundles found per pkg name
	mapPackagesWithBundlesShouldExist := make(map[string][]bundles.Column)
	for _, v := range shouldExist {
		mapPackagesWithBundlesShouldExist[v.PackageName] = append(mapPackagesWithBundlesShouldExist[v.PackageName], v)
	}

	shouldExistResult := []Result{}
	for key, bundles := range mapPackagesWithBundlesShouldExist {
		shouldExistResult = append(shouldExistResult, Result{PackageName: key, Bundles: bundles})
	}

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/deprecated-bundles-repo/validate_community/template.go.tmpl")))
	err = t.Execute(f, File{ShouldExist: shouldExistResult, ShouldNotExist: shouldNotExistResult})
	if err != nil {
		panic(err)
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
