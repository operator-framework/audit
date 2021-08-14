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
// generates the deprecate.yml file with all bundles which requires
// to be deprecated because are using the APIs which will be removed on ocp 4.9 .
package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

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
	Deprecated []Deprecated
}

//nolint: lll
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// Update here the path of the JSON report for the image that you would like to be used
	path := "testdata/reports/redhat_redhat_operator_index/bundles_registry.redhat.io_redhat_redhat_operator_index_v4.8_2021-06-19.json"

	byteValue, err := pkg.ReadFile(filepath.Join(currentPath, path))
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

	// filter by all pkgs which has only deprecated APIs
	hasDeprecated := make(map[string][]bundles.Column)
	for key, bundles := range mapPackagesWithBundles {
		for _, b := range bundles {
			if len(b.KindsDeprecateAPIs) > 0 {
				hasDeprecated[key] = mapPackagesWithBundles[key]
			}
		}
	}

	// create the object with the bundle path
	// see that we need to remove the redhat registry domain
	allDeprecated := []Deprecated{}
	for key, bundles := range hasDeprecated {
		deprecatedYaml := Deprecated{PackageName: key}

		// nolint:scopelint
		sort.Slice(bundles[:], func(i, j int) bool {
			return bundles[i].BundleName < bundles[j].BundleName
		})

		for _, b := range bundles {
			deprecatedYaml.Bundles = append(deprecatedYaml.Bundles,
				Bundles{
					Paths:   strings.ReplaceAll(b.BundleImagePath, "registry.redhat.io/", ""),
					Details: b.BundleName,
				})
		}
		allDeprecated = append(allDeprecated, deprecatedYaml)
	}

	sort.Slice(allDeprecated[:], func(i, j int) bool {
		return allDeprecated[i].PackageName < allDeprecated[j].PackageName
	})

	f, err := os.Create(filepath.Join(currentPath, "hack/scripts/deprecated-bundles-repo/deprecate-all/deprecated.yml"))
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/scripts/deprecated-bundles-repo/deprecate-all/template.go.tmpl")))
	err = t.Execute(f, File{allDeprecated})
	if err != nil {
		panic(err)
	}

}
