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
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type SecurityContextInfo struct {
	Name   string
	ContainersPretty []string
	DeploymentsPretty []string
	Containers []corev1.SecurityContext
	Deployments []corev1.PodSecurityContext
	DeploymentsLabels []string
	BundleLabels []string
	HaveAccessToSCCV2 string
}

type Package struct {
	PackageName string
	Bundles     []SecurityContextInfo
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
				if !strings.Contains(info.Name(), "v4.11") {
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
		pkg.GetReportName(report.ImageName, "security_context", "html"))

	f, err := os.Create(dashOutputPath)
	if err != nil {
		log.Fatal(err)
	}

	t := template.Must(template.ParseFiles(getTemplatePath()))
	err = t.Execute(f, report)
	if err != nil {
		log.Errorf("fail to parse file %s", err)
	}

	return f.Close()
}

func getTemplatePath() string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(currentPath, "/hack/specific-needs/securityContext/template.go.tmpl")
}

func generateOpenshiftNSReport(bundlesReport bundles.Report) *OpenshiftNSReport {
	report := &OpenshiftNSReport{}
	report.ImageName = bundlesReport.Flags.IndexImage
	report.ImageID = bundlesReport.IndexImageInspect.ID
	report.ImageBuild = bundlesReport.IndexImageInspect.Created
	report.GeneratedAt = bundlesReport.GenerateAt

	allBundlesWithInfo := getMapWithRestrictedInfo(bundlesReport)

	for pkgName, bundles := range allBundlesWithInfo {
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

func getMapWithRestrictedInfo(bundlesReport bundles.Report) map[string][]SecurityContextInfo {
	mapPackageBundles := make(map[string][]SecurityContextInfo)

	for _, bundle := range bundlesReport.Columns {
		if bundle.IsDeprecated ||
			len(bundle.PackageName) == 0 ||
			bundle.BundleCSV == nil ||
			bundle.BundleCSV.Annotations == nil {
			continue
		}
		var bundleRestricted SecurityContextInfo
		bundleRestricted.Name = bundle.BundleCSV.Name
		bundleRestricted.HaveAccessToSCCV2 = "LOW"

		// get pod-security labels (let's see if people is doing things wrong)
		for _, label := range bundle.BundleCSV.Labels {
			if strings.HasPrefix(label,"pod-security") {
				bundleRestricted.BundleLabels = append(bundleRestricted.BundleLabels, label)
			}
		}

		dep := bundle.BundleCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs
		if dep != nil {
			for _, value := range dep {
				if value.Spec.Template.Spec.SecurityContext != nil {
					// get the securityContext info
					bundleRestricted.Deployments = append(bundleRestricted.Deployments,
						*value.Spec.Template.Spec.SecurityContext)

					// store the securityContext as pretty info
					pretty, err := yaml.Marshal(value.Spec.Template.Spec.SecurityContext)
					if err != nil {
						log.Warnf(err.Error())
					}
					bundleRestricted.DeploymentsPretty = append(bundleRestricted.DeploymentsPretty,
						fmt.Sprintf("\n%s\n\n", string(pretty)))

					// get pod-security labels
					for _, label := range value.Spec.Template.Labels {
						if strings.HasPrefix(label,"pod-security") {
							bundleRestricted.DeploymentsLabels = append(bundleRestricted.DeploymentsLabels, label)
						}
					}

					if !hasAccessPod(value) {
						bundleRestricted.HaveAccessToSCCV2 = "HIGH"
					}
				}

				if value.Spec.Template.Spec.Containers != nil {
					for _, value := range value.Spec.Template.Spec.Containers {
						if value.SecurityContext != nil {
							bundleRestricted.Containers = append(bundleRestricted.Containers,
								*value.SecurityContext)

							pretty, err := yaml.Marshal(value.SecurityContext)
							if err != nil {
								log.Warnf(err.Error())
							}
							bundleRestricted.ContainersPretty = append(bundleRestricted.ContainersPretty,
								fmt.Sprintf("%s", pretty))

							if !hasAccessContainers(*value.SecurityContext) {
								bundleRestricted.HaveAccessToSCCV2 = "NO (NOT RUN)"
							}
						}
					}
				}
			}
		}
		mapPackageBundles[bundle.PackageName] = append(mapPackageBundles[bundle.PackageName], bundleRestricted)
	}
	return mapPackageBundles
}

func hasAccessPod(value v1alpha1.StrategyDeploymentSpec) bool {
	if value.Spec.Template.Spec.SecurityContext.RunAsNonRoot != nil && !*value.Spec.Template.Spec.SecurityContext.RunAsNonRoot {
		return false
	}
	return true
}

func hasAccessContainers(value corev1.SecurityContext) bool{
	if value.RunAsNonRoot != nil && !*value.RunAsNonRoot{
		return false
	}

	if value.AllowPrivilegeEscalation != nil && *value.AllowPrivilegeEscalation {
		return false
	}

	if value.Privileged != nil && *value.Privileged {
		return false
	}

	if value.Capabilities != nil {
		if len(value.Capabilities.Add) > 0 {
			return false
		}
		var allCapabilty corev1.Capability
		var allCapabilty2 corev1.Capability
		allCapabilty = "all"
		allCapabilty2 = "ALL"
		if value.Capabilities.Drop != nil &&
			(value.Capabilities.Drop[0] != allCapabilty && value.Capabilities.Drop[0] != allCapabilty2){
			return false
		}
	}
	return true
}