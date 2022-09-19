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
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	//nolint: typecheck
	"github.com/goccy/go-yaml"
	"github.com/operator-framework/audit/hack"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type Bundle struct {
	BundleName         string
	ForHideButton      string
	Permissions        string
	ClusterPermissions string
}

type Package struct {
	PackageName string
	Bundles     []Bundle
}

type RBACReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
	Nodes       []Package
	Daemonset   []Package
}

func main() {
	log.Info("Starting ...")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	reportPath := filepath.Join(currentPath, hack.ReportsPath, "rbac")

	command := exec.Command("rm", "-rf", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	custom.Flags.OutputPath = reportPath

	dirs := map[string]string{
		"redhat_certified_operator_index": "registry.redhat.io/redhat/certified-operator-index",
		"redhat_community_operator_index": "registry.redhat.io/redhat/community-operator-index",
		"redhat_redhat_marketplace_index": "registry.redhat.io/redhat/redhat-marketplace-index",
		"redhat_redhat_operator_index":    "registry.redhat.io/redhat/redhat-operator-index",
	}

	allPaths := []string{}
	// nolint:scopelint
	for dir := range dirs {
		pathToWalk := filepath.Join(currentPath, hack.ReportsPath, dir)
		if _, err := os.Stat(pathToWalk); err != nil && os.IsNotExist(err) {
			continue
		}

		// Walk in the testdata dir and generates the deprecated-api custom dashboard for
		// all bundles JSON reports available there
		// nolint:staticcheck
		err := filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasPrefix(info.Name(), "bundles") &&
				strings.HasSuffix(info.Name(), "json") {
				allPaths = append(allPaths, path)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, v := range allPaths {
		custom.Flags.File = v
		err = generateReportFor()
		if err != nil {
			log.Error(err)
		}
	}

	log.Infof("Operation completed.")
}

func generateReportFor() error {
	bundles, err := custom.ParseBundlesJSONReport()
	if err != nil {
		log.Fatal(err)
	}

	report := generateRBACReport(bundles)

	dashOutputPath := filepath.Join(custom.Flags.OutputPath,
		pkg.GetReportName(report.ImageName, "rbac", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(getTemplatePath()))
	err = t.Execute(f, report)
	if err != nil {
		log.Fatal(err)
	}

	return f.Close()
}

func getTemplatePath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(currentPath, "/hack/specific-needs/rbac/template.go.tmpl")
}

func generateRBACReport(bundlesReport bundles.Report) *RBACReport {
	rbacReport := &RBACReport{}
	rbacReport.ImageName = bundlesReport.Flags.IndexImage
	rbacReport.ImageID = bundlesReport.IndexImageInspect.ID
	rbacReport.ImageBuild = bundlesReport.IndexImageInspect.Created
	rbacReport.GeneratedAt = bundlesReport.GenerateAt

	allBundlesWithNode := getAllWithRBACForNodes(bundlesReport)
	allBundlesWithDeamonset := getAllBundlesWithRBACForDeamonset(bundlesReport)

	pkgWithNodes := mapBundlesPerPackage(allBundlesWithNode)
	pkgsWithDeamonset := mapBundlesPerPackage(allBundlesWithDeamonset)

	for pkgName, bundles := range pkgWithNodes {
		nodeBundles := getReportValues(bundles)
		rbacReport.Nodes = append(rbacReport.Nodes, Package{
			PackageName: pkgName,
			Bundles:     nodeBundles,
		})
	}

	for pkgName, bundles := range pkgsWithDeamonset {
		deamonsetBundles := getReportValues(bundles)
		rbacReport.Daemonset = append(rbacReport.Nodes, Package{
			PackageName: pkgName,
			Bundles:     deamonsetBundles,
		})
	}

	return rbacReport
}

func getReportValues(bundlesColum []bundles.Column) []Bundle {
	var bundlesResult []Bundle
	for _, bundle := range bundlesColum {

		perm := ""
		//nolint: typecheck
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions != nil {
			permYAML, err := yaml.Marshal(bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions)
			if err != nil {
				log.Fatalf(err.Error())
			}
			perm = fmt.Sprintf("\n%s\n\n", string(permYAML))
		}

		clusterPerm := ""
		//nolint: typecheck
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions != nil {
			permYAML, err := yaml.Marshal(bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions)
			if err != nil {
				log.Fatalf(err.Error())
			}

			clusterPerm = fmt.Sprintf("\n%s\n\n", string(permYAML))
		}

		namehidden := bundle.BundleCSV.Name
		namehidden = strings.Replace(namehidden, "_", "", -1)
		namehidden = strings.Replace(namehidden, ".", "", -1)
		namehidden = strings.Replace(namehidden, "-", "", -1)
		bundlesResult = append(bundlesResult, Bundle{BundleName: bundle.BundleCSV.Name,
			Permissions:        perm,
			ClusterPermissions: clusterPerm,
			ForHideButton:      namehidden,
		})
	}

	sort.Slice(bundlesResult[:], func(i, j int) bool {
		return bundlesResult[i].BundleName < bundlesResult[j].BundleName
	})

	return bundlesResult
}

func mapBundlesPerPackage(bundlesReport []bundles.Column) map[string][]bundles.Column {
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport {
		if len(v.PackageName) == 0 {
			continue
		}
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}
	return mapPackagesWithBundles
}

// nolint: dupl
func getAllBundlesWithRBACForDeamonset(bundlesReport bundles.Report) []bundles.Column {
	var allBundlesWithDeamonsets []bundles.Column
	for _, bundle := range bundlesReport.Columns {
		found := false
		if bundle.BundleCSV == nil {
			continue
		}
		if bundle.IsDeprecated {
			continue
		}
		if len(bundle.PackageName) == 0 {
			continue
		}
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions != nil {
			for _, perms := range bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions {
				if hasBundleDaemonsetsCriteria(perms) {
					allBundlesWithDeamonsets = append(allBundlesWithDeamonsets, bundle)
					found = true
					break
				}
			}
		}
		if found {
			continue
		}
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions != nil {
			for _, perms := range bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
				if hasBundleDaemonsetsCriteria(perms) {
					allBundlesWithDeamonsets = append(allBundlesWithDeamonsets, bundle)
					break
				}
			}
		}
	}
	return allBundlesWithDeamonsets
}

//nolint:dupl
func getAllWithRBACForNodes(bundlesReport bundles.Report) []bundles.Column {
	var allBundlesWithNode []bundles.Column

	for _, bundle := range bundlesReport.Columns {
		foundNode := false
		if bundle.BundleCSV == nil {
			continue
		}
		if bundle.IsDeprecated {
			continue
		}
		if len(bundle.PackageName) == 0 {
			continue
		}
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions != nil {
			for _, perms := range bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions {
				if hasBundleNodeCriteria(perms) {
					allBundlesWithNode = append(allBundlesWithNode, bundle)
					foundNode = true
					break
				}
			}
		}
		if foundNode {
			continue
		}
		if bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions != nil {
			for _, perms := range bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
				if hasBundleNodeCriteria(perms) {
					allBundlesWithNode = append(allBundlesWithNode, bundle)
					break
				}
			}
		}
	}
	return allBundlesWithNode
}

func hasBundleDaemonsetsCriteria(perms v1alpha1.StrategyDeploymentPermissions) bool {
	if perms.Rules != nil {
		for _, rule := range perms.Rules {
			for _, names := range rule.Resources {
				if names == "daemonsets" {
					if rule.Verbs == nil {
						continue
					}
					for _, verb := range rule.Verbs {
						if checkForWritingPermissions(verb) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func hasBundleNodeCriteria(perms v1alpha1.StrategyDeploymentPermissions) bool {
	if perms.Rules != nil {
		for _, rule := range perms.Rules {
			for _, names := range rule.Resources {
				if names == "nodes" || names == "nodes/status" {
					if rule.Verbs == nil {
						continue
					}
					for _, verb := range rule.Verbs {
						if checkForWritingPermissions(verb) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func checkForWritingPermissions(verb string) bool {
	return verb == "create" ||
		verb == "patch" ||
		verb == "update" ||
		verb == "*"
}
