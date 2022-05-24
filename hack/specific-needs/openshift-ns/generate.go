// Copyright 2022 The Audit Authors
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

// This script when execute look for all bundles index reports generates under
// testdata reports and inside of the each image directory so that can obtain
// the required data to generate its report.
//
// This report looks for list Packages and its bundles which are requesting
// create/update/patch permissions for node and node/status API. Also, this report
// looks for all scenarios which has RBCA permission to create/update/patch deamonsets.
//
// It might be removed in the future or become to be some kind of workflow check.
package main

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/audit/hack"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type BundlesAnnotation struct {
	Name   string
	Values []string
}

type Package struct {
	PackageName string
	Bundles     []BundlesAnnotation
}

type OpenshiftNSReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
	Packages    []Package
}

func main() {
	log.Info("Starting ...")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dirs := map[string]string{
		"redhat_certified_operator_index": "registry.redhat.io/redhat/certified-operator-index",
		"redhat_community_operator_index": "registry.redhat.io/redhat/community-operator-index",
		"redhat_redhat_marketplace_index": "registry.redhat.io/redhat/redhat-marketplace-index",
		"redhat_redhat_operator_index":    "registry.redhat.io/redhat/redhat-operator-index",
	}

	// nolint:scopelint
	for dir := range dirs {
		pathToWalk := filepath.Join(currentPath, hack.ReportsPath, dir)
		if _, err := os.Stat(pathToWalk); err != nil && os.IsNotExist(err) {
			continue
		}

		// Walk in the testdata dir and generates the deprecated-api custom dashboard for
		// all bundles JSON reports available there

		err := filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasPrefix(info.Name(), "bundles") &&
				strings.HasSuffix(info.Name(), "json") {

				// ignore all tags and only generate for 4.10
				if !strings.Contains(info.Name(), "v4.10") {
					return nil
				}

				custom.Flags.OutputPath = filepath.Join(hack.ReportsPath, dir, "dashboards")
				custom.Flags.File = path
				err = generateReportFor()
				if err != nil {
					log.Error(err)
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Infof("Operation completed.")
}

func generateReportFor() error {
	bundles, err := custom.ParseBundlesJSONReport()
	if err != nil {
		log.Fatal(err)
	}

	report := generateOpenshiftNSReport(bundles)

	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(report.ImageName, "openshift-ns", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(getTemplatePath()))
	err = t.Execute(f, report)
	if err != nil {
		panic(err)
	}

	return f.Close()
}

func getTemplatePath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(currentPath, "/hack/specific-needs/openshift-ns/template.go.tmpl")
}

func generateOpenshiftNSReport(bundlesReport bundles.Report) *OpenshiftNSReport {
	report := &OpenshiftNSReport{}
	report.ImageName = bundlesReport.Flags.IndexImage
	report.ImageID = bundlesReport.IndexImageInspect.ID
	report.ImageBuild = bundlesReport.IndexImageInspect.Created
	report.GeneratedAt = bundlesReport.GenerateAt

	allBundlesWithAnnotations := getMapWithAnnotations(bundlesReport)

	for pkgName, bundles := range allBundlesWithAnnotations {
		report.Packages = append(report.Packages, Package{
			PackageName: pkgName,
			Bundles:     bundles,
		})
	}

	sort.Slice(report.Packages[:], func(i, j int) bool {
		return report.Packages[i].PackageName < report.Packages[j].PackageName
	})

	return report
}

func getMapWithAnnotations(bundlesReport bundles.Report) map[string][]BundlesAnnotation {
	mapPackageBundles := make(map[string][]BundlesAnnotation)

	for _, bundle := range bundlesReport.Columns {
		var bundleAnnotationsFoundValues []string
		if bundle.IsDeprecated ||
			len(bundle.PackageName) == 0 ||
			bundle.BundleCSV == nil ||
			bundle.BundleCSV.Annotations == nil {
			continue
		}
		for key, value := range bundle.BundleCSV.Annotations {
			if key == "operatorframework.io/suggested-namespace" && strings.HasPrefix(value, "openshift") {
				bundleAnnotationsFoundValues = append(bundleAnnotationsFoundValues, value)
			}
		}
		if len(bundleAnnotationsFoundValues) > 0 {
			mapPackageBundles[bundle.PackageName] = append(mapPackageBundles[bundle.PackageName], BundlesAnnotation{
				bundle.BundleCSV.Name,
				bundleAnnotationsFoundValues,
			})
		}
	}
	return mapPackageBundles
}
