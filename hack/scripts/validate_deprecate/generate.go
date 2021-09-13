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

	log "github.com/sirupsen/logrus"

	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

//nolint:lll,govet,gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var deprecateFile string
	var image string
	var outputPath string

	flag.StringVar(&outputPath, "output", currentPath, "Inform the path for output the report, if not informed it will be generated at hack/scripts/deprecated-bundles-repo/deprecate-green.")

	flag.StringVar(&deprecateFile, "deprecate", "", "Inform the path with the deprecate json file")
	flag.StringVar(&image, "image", "", "inform the final image result")

	flag.Parse()

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
	file := filepath.Join(outputPath, nameFinalJSONReport)
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

	byteValue, err := pkg.ReadFile(deprecateFile)
	if err != nil {
		log.Fatal(err)
	}

	var deprecatedBundles []string
	if err = json.Unmarshal(byteValue, &deprecatedBundles); err != nil {
		log.Fatal(err)
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

	if len(notDeprecated) == 0 {
		log.Info("WORKED\n Not found any bundle informed to be deprecated that exists in the index or does not have the olm.deprecated property")
	} else {
		log.Errorf("ERROR: Following the bundles that seems not deprecated when should be\n")
		for _, v := range notDeprecated {
			log.Errorf("Bundle Name: (%s) / Bundle Path (%s) \n", v.BundleName, v.BundleImagePath)
		}
	}

	log.Infof("You can check the HTML report for the final result in %", filepath.Join(outputPath, pkg.GetReportName(image, "deprecate-apis", "html")))
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
