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
package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
	log "github.com/sirupsen/logrus"
)

type File struct {
	APIDashReport *custom.APIDashReport
}

//nolint: lll
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	defaultOutputPath := "hack/scripts/packages"

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

	apiDashReport, err := getAPIDashForImage(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	sort.Slice(apiDashReport.PartialComplying[:], func(i, j int) bool {
		return apiDashReport.PartialComplying[i].Name < apiDashReport.PartialComplying[j].Name
	})

	fp := filepath.Join(currentPath, outputPath, pkg.GetReportName(apiDashReport.ImageName, "package", "txt"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/packages/template.go.tmpl")))
	err = t.Execute(f, File{APIDashReport: apiDashReport})
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
