// Copyright 2021 The Audit Authors
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

// This script is a helper to generate index for reports done in the specific needs.
package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/audit/hack"
)

type DashboardPerCatalog struct {
	Name    string
	Reports []Reports
}

type Reports struct {
	Path string
	Name string
	Kind string
}

type Index struct {
	DashboardPerCatalog []DashboardPerCatalog
}

//nolint:gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	//todo: update here the dir to generate the index
	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath, "annotations")

	var all []DashboardPerCatalog
	var index Index

	pathToWalk := filepath.Join(fullReportsPath)

	dash := DashboardPerCatalog{Name: "Annotations"}
	err = filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
		if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), "html") {
			if !strings.Contains(info.Name(), "annotations") {
				return nil
			}

			var kind = "UNKNOWN"
			if strings.Contains(info.Name(), "certified") {
				kind = "Source: Certified"
			} else if strings.Contains(info.Name(), "marketplace") {
				kind = "Source: Marketplace"
			} else if strings.Contains(info.Name(), "community") {
				kind = "Source: Community"
			} else if strings.Contains(info.Name(), "redhat_operator") {
				kind = "Source: RedHat"
			} else if strings.Contains(info.Name(), "maxocp") {
				kind = "Max OCP Version - Monitor"
			}

			tagValue := "latest"
			if strings.Contains(info.Name(), "v") {
				tagS := strings.Split(info.Name(), "v")[1]
				tagValue = strings.Split(tagS, "_")[0]
			}

			tagValue = strings.Replace(tagValue, ".html", "", -1)

			name := fmt.Sprintf("[%s] - Tag: %s", kind, tagValue)
			//nolint scopelint
			dash.Reports = append(dash.Reports,
				Reports{Path: filepath.Join(info.Name()),
					Name: name, Kind: kind})
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(dash.Reports[:], func(i, j int) bool {
		return dash.Reports[i].Name < dash.Reports[j].Name
	})
	all = append(all, dash)

	sort.Slice(all[:], func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	index.DashboardPerCatalog = all

	indexPath := filepath.Join(pathToWalk, "index.html")

	f, err := os.Create(indexPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(filepath.Join(currentPath, "hack/index/template.go.tmpl")))
	err = t.Execute(f, index)
	if err != nil {
		log.Fatalf("error to exec %v", err)
	}
	f.Close()
}
