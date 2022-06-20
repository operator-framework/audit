// Copyright 2022 The Audit Authors
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

// It is a helper script to compare the RedHat vs Community catalog and find
// the content which is from RedHat and exist in both indexes.
package main

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

//nolint:gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath)

	dirs := map[string]string{
		"redhat_community_operator_index": "registry.redhat.io/redhat/community-operator-index",
		"redhat_redhat_operator_index":    "registry.redhat.io/redhat/redhat-operator-index",
	}

	catalogsPath := filepath.Join(fullReportsPath, "vs-catalogs")

	command := exec.Command("rm", "-rf", catalogsPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", catalogsPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	custom.Flags.OutputPath = catalogsPath

	allPaths := map[string]string{}
	// nolint:scopelint
	for dir := range dirs {
		pathToWalk := filepath.Join(fullReportsPath, dir)
		if _, err := os.Stat(pathToWalk); err != nil && os.IsNotExist(err) {
			continue
		}

		// Walk in the testdata dir and generates the deprecated-api custom dashboard for
		// all bundles JSON reports available there

		err := filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasPrefix(info.Name(), "bundles") &&
				strings.HasSuffix(info.Name(), "json") {

				// Ignore the tag images 4.6 and 4.7
				if strings.Contains(info.Name(), "v4.6") {
					allPaths["v4.6"] += fmt.Sprintf(";%s", path)
				} else if strings.Contains(info.Name(), "v4.7") {
					allPaths["v4.7"] += fmt.Sprintf(";%s", path)
				} else if strings.Contains(info.Name(), "v4.8") {
					allPaths["v4.8"] += fmt.Sprintf(";%s", path)
				} else if strings.Contains(info.Name(), "v4.9") {
					allPaths["v4.9"] += fmt.Sprintf(";%s", path)
				} else if strings.Contains(info.Name(), "v4.10") {
					allPaths["v4.10"] += fmt.Sprintf(";%s", path)
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	for k, v := range allPaths {
		custom.Flags.File = v
		name := fmt.Sprintf("OCP %s", k)
		err = generateReportFor(name, v)
		if err != nil {
			log.Error(err)
		}
	}
}

func generateReportFor(name string, filespath string) error {
	custom.Flags.Files = filespath
	allBundlesReport, err := custom.ParseMultiBundlesJSONReport()
	if err != nil {
		return err
	}

	catalogReport := newCatalogReportReport(allBundlesReport, name)

	reportName := strings.ReplaceAll(name, " ", "_")
	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(reportName, "catalog", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(getTemplatePath()))
	err = t.Execute(f, catalogReport)
	if err != nil {
		log.Fatal(err)
	}

	f.Close()
	log.Infof("Operation completed.")
	return nil
}

func getTemplatePath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(currentPath, "/hack/specific-needs/vs-catalog/template.go.tmpl")
}

type CommunityFounds struct {
	CommunityPackageName string
	HasSameIcon          bool
	HasSameDisplayName   bool
	HasSameKind          bool
	HasSameAPIName       bool
	HasSamePackageName   bool
	HasAPIConflicts      bool
	Kinds                map[string][]string
	APIName              map[string][]string
	APINameVersion       map[string][]string
}

type RedHadPackages struct {
	PackageName     string
	UsesRedHatLogo  bool
	CommunityFounds []CommunityFounds
}

type CatalogReport struct {
	ImageNames     []string
	RedHadPackages []RedHadPackages
	GeneratedAt    string
	FilterPkg      string
	Name           string
}

// nolint:lll
const redhatLogo = "PHN2ZyBpZD0iTGF5ZXJfMSIgZGF0YS1uYW1lPSJMYXllciAxIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxOTIgMTQ1Ij48ZGVmcz48c3R5bGU+LmNscy0xe2ZpbGw6I2UwMDt9PC9zdHlsZT48L2RlZnM+PHRpdGxlPlJlZEhhdC1Mb2dvLUhhdC1Db2xvcjwvdGl0bGU+PHBhdGggZD0iTTE1Ny43Nyw2Mi42MWExNCwxNCwwLDAsMSwuMzEsMy40MmMwLDE0Ljg4LTE4LjEsMTcuNDYtMzAuNjEsMTcuNDZDNzguODMsODMuNDksNDIuNTMsNTMuMjYsNDIuNTMsNDRhNi40Myw2LjQzLDAsMCwxLC4yMi0xLjk0bC0zLjY2LDkuMDZhMTguNDUsMTguNDUsMCwwLDAtMS41MSw3LjMzYzAsMTguMTEsNDEsNDUuNDgsODcuNzQsNDUuNDgsMjAuNjksMCwzNi40My03Ljc2LDM2LjQzLTIxLjc3LDAtMS4wOCwwLTEuOTQtMS43My0xMC4xM1oiLz48cGF0aCBjbGFzcz0iY2xzLTEiIGQ9Ik0xMjcuNDcsODMuNDljMTIuNTEsMCwzMC42MS0yLjU4LDMwLjYxLTE3LjQ2YTE0LDE0LDAsMCwwLS4zMS0zLjQybC03LjQ1LTMyLjM2Yy0xLjcyLTcuMTItMy4yMy0xMC4zNS0xNS43My0xNi42QzEyNC44OSw4LjY5LDEwMy43Ni41LDk3LjUxLjUsOTEuNjkuNSw5MCw4LDgzLjA2LDhjLTYuNjgsMC0xMS42NC01LjYtMTcuODktNS42LTYsMC05LjkxLDQuMDktMTIuOTMsMTIuNSwwLDAtOC40MSwyMy43Mi05LjQ5LDI3LjE2QTYuNDMsNi40MywwLDAsMCw0Mi41Myw0NGMwLDkuMjIsMzYuMywzOS40NSw4NC45NCwzOS40NU0xNjAsNzIuMDdjMS43Myw4LjE5LDEuNzMsOS4wNSwxLjczLDEwLjEzLDAsMTQtMTUuNzQsMjEuNzctMzYuNDMsMjEuNzdDNzguNTQsMTA0LDM3LjU4LDc2LjYsMzcuNTgsNTguNDlhMTguNDUsMTguNDUsMCwwLDEsMS41MS03LjMzQzIyLjI3LDUyLC41LDU1LC41LDc0LjIyYzAsMzEuNDgsNzQuNTksNzAuMjgsMTMzLjY1LDcwLjI4LDQ1LjI4LDAsNTYuNy0yMC40OCw1Ni43LTM2LjY1LDAtMTIuNzItMTEtMjcuMTYtMzAuODMtMzUuNzgiLz48L3N2Zz4="

// nolint:gocyclo
func newCatalogReportReport(bundlesReport []bundles.Report, name string) *CatalogReport {
	catalogReport := CatalogReport{}
	dt := time.Now().Format("2006-01-02")
	catalogReport.GeneratedAt = dt
	catalogReport.Name = name

	// Get image names
	for _, bR := range bundlesReport {
		catalogReport.ImageNames = append(catalogReport.ImageNames, bR.Flags.IndexImage)
	}

	// First check all from community
	var redhatIndexReport bundles.Report
	for _, bR := range bundlesReport {

		catalogName := getCatalogIndexName(bR.Flags.IndexImage)
		if catalogName != "RedHat Index" {
			continue
		}

		redhatIndexReport = bR
	}

	var communityIndexReport bundles.Report
	for _, bR := range bundlesReport {

		catalogName := getCatalogIndexName(bR.Flags.IndexImage)
		if catalogName != "RedHat Community Index" {
			continue
		}
		communityIndexReport = bR
	}

	var allRedHat []RedHadPackages
	for _, bundleRedhat := range redhatIndexReport.Columns {
		if len(bundleRedhat.PackageName) == 0 {
			continue
		}

		rhPkg := RedHadPackages{}
		rhPkg.PackageName = bundleRedhat.PackageName
		for _, icon := range bundleRedhat.BundleCSV.Spec.Icon {
			if icon.Data == redhatLogo {
				rhPkg.UsesRedHatLogo = true
			}
		}

		found := false
		index := 0
		for i, redHatPkg := range allRedHat {
			if redHatPkg.PackageName == bundleRedhat.PackageName {
				rhPkg = redHatPkg
				found = true
				index = i
				break
			}
		}
		var allCommunityFounds []CommunityFounds

		for _, bundleCommunity := range communityIndexReport.Columns {
			if len(bundleCommunity.PackageName) == 0 {
				continue
			}

			cmF := CommunityFounds{}
			cmF.CommunityPackageName = bundleCommunity.PackageName
			foundC := false
			indexC := 0
			for i, allCFounds := range allCommunityFounds {
				if allCFounds.CommunityPackageName == bundleCommunity.PackageName {
					cmF = allCFounds
					foundC = true
					indexC = i
					break
				}
			}

			if bundleCommunity.BundleCSV == nil {
				continue
			}

			if cmF.Kinds == nil {
				cmF.Kinds = map[string][]string{}
			}

			if cmF.APIName == nil {
				cmF.APIName = map[string][]string{}
			}

			if cmF.APINameVersion == nil {
				cmF.APINameVersion = map[string][]string{}
			}

			// check if has the same pkg name
			if bundleCommunity.PackageName == bundleRedhat.PackageName {
				cmF.HasSamePackageName = true
				cmF.CommunityPackageName = bundleCommunity.PackageName
			}

			if len(bundleCommunity.BundleCSV.Spec.Icon) > 0 && len(bundleRedhat.BundleCSV.Spec.Icon) > 0 {
				if hasSameIcons(bundleCommunity.BundleCSV, bundleRedhat.BundleCSV) {
					cmF.HasSameIcon = true
				}
			}

			if strings.TrimSpace(bundleCommunity.BundleCSV.Spec.DisplayName) ==
				strings.TrimSpace(bundleRedhat.BundleCSV.Spec.DisplayName) {
				cmF.HasSameIcon = true
			}

			for _, crdRedHat := range bundleRedhat.BundleCSV.Spec.CustomResourceDefinitions.Owned {
				for _, crdCommunity := range bundleCommunity.BundleCSV.Spec.CustomResourceDefinitions.Owned {
					if crdRedHat.Kind == crdCommunity.Kind {
						cmF.HasSameKind = true
					}

					if crdRedHat.Name == crdCommunity.Name {
						cmF.HasSameAPIName = true
					}

					redHatAPI := fmt.Sprintf("%s/%s", crdRedHat.Name, crdRedHat.Version)
					if redHatAPI == fmt.Sprintf("%s/%s", crdCommunity.Name, crdCommunity.Version) {
						cmF.HasAPIConflicts = true
					}

					if redHatAPI == fmt.Sprintf("%s/%s", crdCommunity.Name, crdCommunity.Version) {
						cmF.APINameVersion[redHatAPI] = append(cmF.APINameVersion[redHatAPI], "")
					} else if crdRedHat.Name == crdCommunity.Name {
						cmF.APIName[crdRedHat.Name] = append(cmF.APIName[crdCommunity.Name], "")
					} else if crdRedHat.Kind == crdCommunity.Kind {
						cmF.Kinds[crdCommunity.Kind] = append(cmF.Kinds[crdCommunity.Kind], "")
					}
				}
			}

			if cmF.HasSameIcon ||
				cmF.HasSameDisplayName ||
				cmF.HasSameKind ||
				cmF.HasSameAPIName ||
				cmF.HasSamePackageName ||
				cmF.HasAPIConflicts {

				// ignore only == kind scenario
				if !cmF.HasSameIcon &&
					!cmF.HasSameDisplayName &&
					cmF.HasSameKind &&
					!cmF.HasSameAPIName &&
					!cmF.HasSamePackageName &&
					!cmF.HasAPIConflicts {
					continue
				}

				if !foundC {
					allCommunityFounds = append(allCommunityFounds, cmF)
				} else {
					allCommunityFounds[indexC] = cmF
				}
			}
		}

		rhPkg.CommunityFounds = allCommunityFounds

		if !found {
			allRedHat = append(allRedHat, rhPkg)
		} else {
			allRedHat[index] = rhPkg
		}
	}

	onlyWithFounds := []RedHadPackages{}
	for _, v := range allRedHat {
		if len(v.CommunityFounds) > 0 {
			onlyWithFounds = append(onlyWithFounds, v)
		}
	}

	catalogReport.RedHadPackages = onlyWithFounds

	return &catalogReport
}

func hasSameIcons(csvA, csvB *v1alpha1.ClusterServiceVersion) bool {
	for _, iconA := range csvA.Spec.Icon {
		for _, iconB := range csvB.Spec.Icon {
			if iconA.Data != redhatLogo && len(iconA.Data) > 0 && len(iconB.Data) > 0 &&
				iconA.Data == iconB.Data && iconA.MediaType == iconB.MediaType {
				return true
			}
		}
	}
	return false
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
