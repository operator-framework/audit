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
// This script is only a helper for we are able to compare to JSON files
// and check what packages were defined in one and are no longer in the other
// one. E.g After send a test to iib I want to know what packages in green
// were in the JSON A which are no longer in its result JSON B
// todo: remove after 4.9-GA

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/operator-framework/audit/hack"
	log "github.com/sirupsen/logrus"

	"os"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type File struct {
	UnableToReAdd []bundles.Column
	NotDeprecated []bundles.Column
}

//nolint:lll,govet,gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var deprecateFile string
	var image string
	var tag string
	var outputPath string

	flag.StringVar(&outputPath, "output", currentPath, "Inform the path for output the report, if not informed it will be generated at hack/scripts/deprecated-bundles-repo/deprecate-green.")

	flag.StringVar(&deprecateFile, "deprecate", "", "Inform the path with the deprecate json file")
	flag.StringVar(&image, "image", "", "inform the final image result")
	flag.StringVar(&tag, "tag", "", "inform the tag value to test opm")

	flag.Parse()
	file := "testdata/reports/deprecate-green/deprecate-green_registry.redhat.io_redhat_redhat_operator_index_v4.9_2021-09-13.json"
	byteValue, err := pkg.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	var deprecatedBundles []string
	if err = json.Unmarshal(byteValue, &deprecatedBundles); err != nil {
		log.Fatal(err)
	}

	binPath := filepath.Join(currentPath, hack.BinPath)
	command := exec.Command(binPath, "index", "bundles",
		fmt.Sprintf("--index-image=%s", image),
		"--output=json",
		"--disable-scorecard",
		fmt.Sprintf("--output-path=%s", outputPath),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	nameFinalJSONReport := pkg.GetReportName(image, "bundles", "json")
	file = filepath.Join(outputPath, nameFinalJSONReport)
	apiDashReportFinalResult, err := getAPIDashForImage(file)
	if err != nil {
		log.Fatal(err)
	}

	command = exec.Command(binPath, "dashboard", "deprecate-apis",
		fmt.Sprintf("--file=%s", file),
		fmt.Sprintf("--output-path=%s", outputPath),
	)

	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	// Get all and check if has any bundle that was configured to be deprecated
	// that was not.
	var notDeprecated []bundles.Column
	for _, v := range apiDashReportFinalResult.Migrated {
		for _, i := range v.AllBundles {
			for _, setToDeprecate := range deprecatedBundles {
				if i.BundleImagePath == setToDeprecate && !i.IsDeprecated {
					notDeprecated = append(notDeprecated, i)
					break
				}
			}
		}
	}
	for _, v := range apiDashReportFinalResult.NotMigrated {
		for _, i := range v.AllBundles {
			for _, setToDeprecate := range deprecatedBundles {
				if i.BundleImagePath == setToDeprecate && !i.IsDeprecated {
					notDeprecated = append(notDeprecated, i)
					break
				}
			}
		}
	}

	// create a map with all bundles found per pkg name
	migratedPkgs := make(map[string][]bundles.Column)
	for _, v := range apiDashReportFinalResult.Migrated {
		for _, b := range v.AllBundles {
			migratedPkgs[b.PackageName] = append(migratedPkgs[b.PackageName], b)
		}
	}

	var unableToReAdd []bundles.Column
	for k, v := range migratedPkgs {
		log.Infof("testing for the package %s", k)
		headOfChannels := custom.GetHeadOfChannels(v)
		for _, head := range headOfChannels {
			log.Infof("testing to re-add with —overwrite-latest the bundle %s with path (%s)", head.BundleName, head.BundleImagePath)
			command = exec.Command("sudo", "opm", "index", "add",
				"--build-tool=docker",
				fmt.Sprintf("--from-index=%s", image),
				fmt.Sprintf("--tag=%s", tag),
				fmt.Sprintf("--bundles='%s'", head.BundleImagePath),
				"--—overwrite-latest",
			)

			_, err := pkg.RunCommand(command)
			if err != nil {
				log.Errorf("running command :%s", err)
				unableToReAdd = append(unableToReAdd, head)
			}
		}
	}

	log.Infof("You can check the HTML report for the final result in %", filepath.Join(outputPath, pkg.GetReportName(image, "deprecate-apis", "html")))

	fp := filepath.Join(outputPath, pkg.GetReportName(image, "validate", "yml"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/deprecated-bundles-repo/validate_deprecate/template.go.tmpl")))
	err = t.Execute(f, File{NotDeprecated: notDeprecated, UnableToReAdd: unableToReAdd})
	if err != nil {
		panic(err)
	}

	defer f.Close()
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
