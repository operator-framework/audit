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
	"fmt"
	"strings"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	log "github.com/sirupsen/logrus"
)

type MultiArchBundle struct {
	HasDisconnectAnnotation bool
	// InfraLabels store the labels
	// "operatorframework.io/arch.amd64": "supported",
	// "operatorframework.io/arch.ppc64le": "supported",
	// "operatorframework.io/arch.s390x": "supported"
	InfraLabels []string
	// Images versus manifest arch
	RelateImages map[string][]string
	// Images versus manifest arch
	InstallImages      map[string][]string
	BundleData         bundles.Column
	Validations        []string
	Supported          map[string]string
	HasMultArchSupport bool
	HasAMD64Support    bool
	HasPPC64leSupport  bool
	HasARM64Support    bool
	Has390xSupport     bool
}

type MultiArchPkg struct {
	Name    string
	Bundles []MultiArchBundle
}

type MultiArchReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
	Packages    []MultiArchPkg
}

// nolint:dupl
func NewMultiArchReport(bundlesReport bundles.Report, filter string) *MultiArchReport {
	multiArch := MultiArchReport{}
	multiArch.ImageName = bundlesReport.Flags.IndexImage
	multiArch.ImageID = bundlesReport.IndexImageInspect.ID
	multiArch.ImageBuild = bundlesReport.IndexImageInspect.Created
	multiArch.GeneratedAt = bundlesReport.GenerateAt

	mapPerPkgHeadsOnly := mapHeadBundlesPerPackageWith(bundlesReport.Columns)
	mapPackagesWithMultData := make(map[string][]MultiArchBundle)

	for pkg, bundles := range mapPerPkgHeadsOnly {
		for _, bundle := range bundles {
			// filter by the name
			if len(filter) > 0 {
				if !strings.Contains(bundle.PackageName, filter) {
					continue
				}
			}
			mb := MultiArchBundle{BundleData: bundle}
			mb.addInfraLabels()
			mb.addDataFromInstallImages(bundlesReport)
			mb.addDataFromRelateImages(bundlesReport)
			mb.checkSupport()
			if len(mb.Supported) > 1 {
				mb.HasMultArchSupport = true
			}
			if len(mb.Supported["amd64"]) > 0 {
				mb.HasAMD64Support = true
			}
			if len(mb.Supported["arm64"]) > 0 {
				mb.HasARM64Support = true
			}
			if len(mb.Supported["ppc64le"]) > 0 {
				mb.HasPPC64leSupport = true
			}
			if len(mb.Supported["s390x"]) > 0 {
				mb.Has390xSupport = true
			}
			mb.validate()
			mapPackagesWithMultData[pkg] = append(mapPackagesWithMultData[pkg], mb)
		}
	}

	for pkg, bundles := range mapPackagesWithMultData {
		multiArch.Packages = append(multiArch.Packages, MultiArchPkg{Name: pkg, Bundles: bundles})
	}
	return &multiArch
}

func (mb *MultiArchBundle) addInfraLabels() {
	for k, v := range mb.BundleData.BundleCSV.ObjectMeta.Labels {
		if strings.Contains(k, "arch") && v == "supported" {
			mb.InfraLabels = append(mb.InfraLabels, k)
		}
	}
}

func (mb *MultiArchBundle) addDataFromInstallImages(bundlesReport bundles.Report) {
	mb.InstallImages = make(map[string][]string)
	if mb.BundleData.BundleCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs != nil {
		for _, v := range mb.BundleData.BundleCSV.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
			for _, c := range v.Spec.Template.Spec.Containers {
				manifest, err := pkg.RunDockerManifestInspect(c.Image, bundlesReport.Flags.ContainerEngine)
				if err != nil {
					// Try again
					if manifest, err = pkg.RunDockerManifestInspect(c.Image, bundlesReport.Flags.ContainerEngine); err != nil {
						mb.BundleData.AuditErrors = append(mb.BundleData.AuditErrors, err.Error())
						log.Errorf("unable to inspect manifests for the container image (%s) : %s", c.Image, err)
						continue
					}
				}
				for _, manifest := range manifest.ManifestData {
					mb.InstallImages[c.Image] = append(mb.InstallImages[c.Image],
						fmt.Sprintf("%s.%s", manifest.Platform.SO,
							manifest.Platform.Architecture))
				}
			}
		}
	}
}

func (mb *MultiArchBundle) addDataFromRelateImages(bundlesReport bundles.Report) {
	mb.RelateImages = make(map[string][]string)
	for _, v := range mb.BundleData.BundleCSV.Spec.RelatedImages {
		manifest, err := pkg.RunDockerManifestInspect(v.Image, bundlesReport.Flags.ContainerEngine)
		if err != nil {
			// Try again
			if manifest, err = pkg.RunDockerManifestInspect(v.Image, bundlesReport.Flags.ContainerEngine); err != nil {
				mb.BundleData.AuditErrors = append(mb.BundleData.AuditErrors, err.Error())
				msg := fmt.Sprintf("unable to inspect manifests for the image (%s) : %s", v.Image, err)
				log.Errorf(msg)
				mb.Validations = append(mb.Validations, msg)
				continue
			}
		}
		if manifest.ManifestData != nil {
			for _, manifest := range manifest.ManifestData {
				mb.RelateImages[v.Image] = append(mb.RelateImages[v.Image],
					fmt.Sprintf("%s.%s", manifest.Platform.SO,
						manifest.Platform.Architecture))
			}
		}
	}
}

func (mb *MultiArchBundle) validate() {
	mb.checkLabels()
	mb.checkValidLabels()
	mb.checkMissingArchtype()
}

// check if any image is missing some archetype
func (mb *MultiArchBundle) checkMissingArchtype() {
	if mb.HasMultArchSupport {
		for image, arc := range mb.RelateImages {
			notFound := []string{}
			for su := range mb.Supported {
				found := false
				for _, t := range arc {
					if strings.Contains(t, su) {
						found = true
						break
					}
				}
				if !found {
					notFound = append(notFound, su)
				}
			}
			if len(notFound) > 0 {
				mb.Validations = append(mb.Validations,
					fmt.Errorf("[bundle %s]: related image (%s) is missing manifest archetype for %q",
						mb.BundleData.BundleCSV.Name, image, notFound).Error())
			}
		}

		for image, arc := range mb.InstallImages {
			notFound := []string{}
			for su := range mb.Supported {
				found := false
				for _, t := range arc {
					if strings.Contains(t, su) {
						found = true
						break
					}
				}
				if !found {
					notFound = append(notFound, su)
				}
			}
			if len(notFound) > 0 {
				mb.Validations = append(mb.Validations,
					fmt.Errorf("[bundle %s]: install image (%s) is missing manifest archetype for %q",
						mb.BundleData.BundleCSV.Name, image, notFound).Error())
			}
		}
	}
}

func (mb *MultiArchBundle) checkValidLabels() {
	supportedArchs := []string{"amd64", "ppc64le", "arm64", "s390x"}
	notOK := []string{}
	for _, infra := range mb.InfraLabels {
		ok := false
		for _, arch := range supportedArchs {
			label := fmt.Sprintf("operatorframework.io/arch.%s", arch)
			if infra == label {
				ok = true
				break
			}
		}
		if !ok {
			notOK = append(notOK, infra)
		}
	}
	if len(notOK) > 0 {
		mb.Validations = append(mb.Validations,
			fmt.Errorf("[bundle %s]: invalid labels: %q", mb.BundleData.BundleCSV.Name, notOK).Error())
	}
}

func (mb *MultiArchBundle) checkLabels() {
	notFoundLabel := []string{}
	if mb.HasMultArchSupport {
		for supported := range mb.Supported {
			found := false
			for _, infra := range mb.InfraLabels {
				if strings.Contains(infra, supported) {
					found = true
					break
				}
			}
			if !found {
				notFoundLabel = append(notFoundLabel, supported)
			}
		}

		if len(notFoundLabel) > 0 {
			mb.Validations = append(mb.Validations,
				fmt.Errorf("[bundle %s]: missing label for %q", mb.BundleData.BundleCSV.Name, notFoundLabel).Error())
		}
	}
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func mapHeadBundlesPerPackageWith(bundlesReport []bundles.Column) map[string][]bundles.Column {
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport {
		if v.IsHeadOfChannel {
			mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
		}
	}
	return mapPackagesWithBundles
}

func (mb *MultiArchBundle) checkSupport() {
	if mb.Supported == nil {
		mb.Supported = make(map[string]string)
	}
	for _, v := range mb.InfraLabels {
		label := strings.ReplaceAll(v, "operatorframework.io/arch.", "")
		mb.Supported[label] = label
	}

	for _, v := range mb.RelateImages {
		for _, soplataform := range v {
			if len(soplataform) > 0 {
				if strings.Contains(soplataform, ".") {
					mb.Supported[strings.Split(soplataform, ".")[1]] = strings.Split(soplataform, ".")[1]
				} else {
					mb.Supported[soplataform] = soplataform
				}
			}
		}
	}

	for _, v := range mb.InstallImages {
		for _, soplataform := range v {
			if len(soplataform) > 0 {
				if strings.Contains(soplataform, ".") {
					mb.Supported[strings.Split(soplataform, ".")[1]] = strings.Split(soplataform, ".")[1]
				} else {
					mb.Supported[soplataform] = soplataform
				}
			}
		}
	}
}
