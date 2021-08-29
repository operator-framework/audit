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
// generates the output.yml file which has all packages which still
// are without a head of channel compatible with 4.9.
// The idea is provide a helper to allow to send emails to notify their authors
// Example of usage: (see that we leave makefile target to help you out here)
// nolint: lll
// go run hack/scripts/ivs-emails/generate.go --mongo=mongo-query-join-results-prod.json --image=testdata/reports/redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.8_2021-08-10.json
// go run hack/scripts/ivs-emails/generate.go --mongo=mongo-query-join-results-prod.json --image=testdata/reports/redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8_2021-08-06.json

package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type MongoContacts struct {
	Type  string `json:"type"`
	Email string `json:"email_address"`
}

type CertProject struct {
	Contacts []MongoContacts `json:"contacts,omitempty"`
}

type MongoItems struct {
	Association string        `json:"association"`
	PackageName string        `json:"package_name"`
	CertProject []CertProject `json:"cert_project,omitempty"`
}

type Item struct {
	MongoItem MongoItems
	CSVEmails []string
	CSVLinks  []string
}

type ImageData struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
}

type Output struct {
	Items     []Item
	ImageData ImageData
}

//nolint: lll
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var mongoFile string
	var jsonFile string

	flag.StringVar(&mongoFile, "mongo", "", "Inform the path for the mongo file with the reqqured data to generate the file. ")
	flag.StringVar(&jsonFile, "image", "", "Inform the path for the JSON result which will be used to generate the report. ")

	flag.Parse()

	byteValue, err := pkg.ReadFile(filepath.Join(currentPath, mongoFile))
	if err != nil {
		log.Fatal(err)
	}

	var mongoValues []MongoItems
	if err = json.Unmarshal(byteValue, &mongoValues); err != nil {
		log.Fatal(err)
	}
	var result Output

	items, image := getData(filepath.Join(currentPath, jsonFile), mongoValues)
	result.Items = items
	result.ImageData = image

	fp := filepath.Join(currentPath, pkg.GetReportName(result.ImageData.ImageName, "ivs", "json"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = hack.ReplaceInFile(fp, "", string(jsonResult))
	if err != nil {
		log.Fatal(err)
	}

}

func getData(image string, mongoValues []MongoItems) ([]Item, ImageData) {
	apiDashReport, err := getAPIDashForImage(image)
	if err != nil {
		log.Fatal(err)
	}

	var items []Item
	for _, pkgV := range apiDashReport.PartialComplying {
		mg := MongoItems{PackageName: pkgV.Name, Association: "N/A"}
		for _, m := range mongoValues {
			if m.PackageName == pkgV.Name {
				mg = m
				break
			}
		}
		var emails []string
		var links []string
		for _, v := range pkgV.AllBundles {
			emails = append(emails, v.MaintainersEmail...)
			links = append(links, v.Links...)
		}
		emails = pkg.GetUniqueValues(emails)
		links = pkg.GetUniqueValues(links)

		items = append(items, Item{MongoItem: mg, CSVEmails: emails, CSVLinks: links})
	}

	sort.Slice(items[:], func(i, j int) bool {
		return items[i].MongoItem.PackageName < items[j].MongoItem.PackageName
	})

	var imageData ImageData

	imageData.ImageName = apiDashReport.ImageName
	imageData.ImageBuild = apiDashReport.ImageBuild
	imageData.ImageID = apiDashReport.ImageID
	imageData.GeneratedAt = apiDashReport.GeneratedAt

	return items, imageData
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
