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
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

//nolint:lll
func main() {

	jsonFinalResult := "my-test.json"
	jsonOrigin := "testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8_2021-08-29.json"

	apiDashReportOrigin, err := getAPIDashForImage(jsonOrigin)
	if err != nil {
		log.Fatal(err)
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	file := filepath.Join(currentPath, jsonFinalResult)

	apiDashReportFinalResult, err := getAPIDashForImage(file)
	if err != nil {
		log.Fatal(err)
	}

	// Check all that was in green that is no longer in the final result
	var notFound []custom.OK
	for _, v := range apiDashReportOrigin.OK {
		found := false
		for _, i := range apiDashReportFinalResult.OK {
			if v.Name == i.Name {
				found = true
				break
			}
		}
		if !found {
			notFound = append(notFound, v)
		}
	}

	reportPath := filepath.Join(currentPath)

	// Creates the compare.json with all packages and bundles data that were
	// in the origin and are no longer in the result
	fp := filepath.Join(reportPath, "compare.json")
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(notFound, "", "\t")
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
