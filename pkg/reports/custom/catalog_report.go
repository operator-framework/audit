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

package custom

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	//log "github.com/sirupsen/logrus"
	"strings"
	"time"

	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type ByGKV struct {
	PackageNames []string
	Catalogs     []string
	GKV          []string
}

type ByPackageName struct {
	PackageName string
	Catalogs    []string
}

type CatalogReport struct {
	ImageNames              []string
	ByPropertiesGKV         []ByGKV
	ByPropertiesPackageName []ByPackageName
	GeneratedAt             string
	FilterPkg               string
	Name                    string
}

// nolint:dupl
func NewCatalogReportReport(bundlesReport []bundles.Report, filterPkg, name string) *CatalogReport {
	catalogReport := CatalogReport{}
	dt := time.Now().Format("2006-01-02")
	catalogReport.GeneratedAt = dt
	catalogReport.FilterPkg = filterPkg
	catalogReport.Name = name

	// Get image names
	for _, bR := range bundlesReport {
		catalogReport.ImageNames = append(catalogReport.ImageNames, bR.Flags.IndexImage)
	}

	catalogReport.addByPropertiesPackageName(bundlesReport)
	catalogReport.addByPropertiesGKV(bundlesReport)

	return &catalogReport
}

type OlmPackage struct {
	PackageName string
	Version     string
}

func (catalogReport *CatalogReport) addByPropertiesPackageName(bundlesReport []bundles.Report) {
	// ByPackageName:
	all := map[string][]string{}
	for _, bR := range bundlesReport {
		for _, bundles := range bR.Columns {
			// filter by the name
			if len(catalogReport.FilterPkg) > 0 {
				if !strings.Contains(bundles.PackageName, catalogReport.FilterPkg) {
					continue
				}
			}

			if bundles.IsDeprecated {
				continue
			}

			if bundles.PackageName == "" || len(strings.TrimSpace(bundles.PackageName)) == 0 {
				continue
			}

			catalogName := getCatalogIndexName(bR.Flags.IndexImage)

			var olmPackage OlmPackage
			var packageName string

			for _, v := range bundles.PropertiesFromDB {
				if v.Type == "olm.package" {
					if err := json.Unmarshal([]byte(v.Value), &olmPackage); err != nil {
						log.Errorf("unable to Unmarshal manifest.json: %s", err)
					}
					packageName = olmPackage.PackageName
					break
				}
			}
			found := false
			for _, list := range all[packageName] {
				if list == catalogName {
					found = true
					break
				}
			}
			if !found {
				all[packageName] = append(all[packageName], catalogName)
			}
		}
	}

	var allItems []ByPackageName
	for pkg, cataList := range all {
		if len(cataList) > 1 {
			allItems = append(allItems, ByPackageName{PackageName: pkg, Catalogs: cataList})
		}
	}

	catalogReport.ByPropertiesPackageName = allItems
}

//nolint gocyclo
func (catalogReport *CatalogReport) addByPropertiesGKV(bundlesReport []bundles.Report) {
	// ByPackageName:
	all := map[string][]string{}
	for _, bR := range bundlesReport {
		for _, bundles := range bR.Columns {
			// filter by the name
			if len(catalogReport.FilterPkg) > 0 {
				if !strings.Contains(bundles.PackageName, catalogReport.FilterPkg) {
					continue
				}
			}

			if bundles.IsDeprecated {
				continue
			}

			if bundles.PackageName == "" || len(strings.TrimSpace(bundles.PackageName)) == 0 {
				continue
			}

			catalogName := getCatalogIndexName(bR.Flags.IndexImage)

			for _, v := range bundles.PropertiesFromDB {
				if v.Type == "olm.gvk" {
					found := false
					for _, list := range all[v.Value] {
						if list == catalogName {
							found = true
							break
						}
					}
					if !found {
						all[v.Value] = append(all[v.Value], catalogName)
					}
				}
			}
		}
	}

	allPerPkg := map[string][]string{}
	for gkv, cataList := range all {
		if len(cataList) > 1 {
			for _, bR := range bundlesReport {
				for _, bundles := range bR.Columns {
					if bundles.PackageName == "" || len(strings.TrimSpace(bundles.PackageName)) == 0 {
						continue
					}
					for _, v := range bundles.PropertiesFromDB {
						if v.Type == "olm.gvk" && v.Value == gkv {
							found := false
							for _, list := range allPerPkg[gkv] {
								if list == bundles.PackageName {
									found = true
									break
								}
							}
							if !found {
								allPerPkg[v.Value] = append(allPerPkg[gkv], bundles.PackageName)
							}
						}
					}
				}
			}
		}
	}

	var allItems []ByGKV
	for gkv, packages := range allPerPkg {
		found := false
		indexList := 0
		for index, list := range allItems {
			for _, pkg := range list.PackageNames {
				for _, pkgFromGKV := range packages {
					if pkgFromGKV == pkg {
						found = true
						indexList = index
						break
					}
				}
			}
		}
		if found {
			allItems[indexList].GKV = append(allItems[indexList].GKV, gkv)
		} else {
			allItems = append(allItems, ByGKV{GKV: []string{gkv}, Catalogs: all[gkv], PackageNames: allPerPkg[gkv]})
		}
	}
	catalogReport.ByPropertiesGKV = allItems
}

func getCatalogIndexName(value string) string {
	if strings.Contains(value, "redhat-operator-index") {
		return "RedHat Index"
	} else if strings.Contains(value, "redhat-marketplace-index") {
		return "Marketplace Index"
	} else if strings.Contains(value, "community-operator-index") {
		return "RedHat Community Index"
	} else if strings.Contains(value, "certified-operator-index") {
		return "Certified Index"
	} else if strings.Contains(value, "operatorhubio") {
		return "OperatorHub.io Index"
	} else if strings.Contains(value, "okd") {
		return "OKD Index"
	} else {
		return value
	}
}
