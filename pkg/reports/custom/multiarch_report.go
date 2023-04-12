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

	log "github.com/sirupsen/logrus"

	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	validator "github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/api/pkg/validation/errors"

	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type MultipleArchitecturesBundleReport struct {
	BundleData          bundles.Column
	InfraLabelsUsed     []string
	AllArchFound        map[string]string
	AllOsFound          map[string]string
	Errors              []string
	Warnings            []string
	ManagerImage        []string
	Images              []string
	HasMultiArchSupport bool
	ForHideButton       string
}

type MultipleArchitecturesPackageReport struct {
	Name    string
	Bundles []MultipleArchitecturesBundleReport
}

type MultipleArchitecturesReport struct {
	ImageName             string
	ImageID               string
	ImageHash             string
	ImageBuild            string
	GeneratedAt           string
	Unsupported           []MultipleArchitecturesPackageReport
	Supported             []MultipleArchitecturesPackageReport
	SupportedWithErrors   []MultipleArchitecturesPackageReport
	SupportedWithWarnings []MultipleArchitecturesPackageReport
}

// platform store the Architecture and OS supported by the image
type platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// nolint:dupl
func NewMultipleArchitecturesReport(bundlesReport bundles.Report, filter,
	containerTool string) *MultipleArchitecturesReport {
	multiArch := MultipleArchitecturesReport{}
	multiArch.ImageName = bundlesReport.Flags.IndexImage
	multiArch.ImageID = bundlesReport.IndexImageInspect.ID
	multiArch.ImageBuild = bundlesReport.IndexImageInspect.Created
	multiArch.GeneratedAt = bundlesReport.GenerateAt

	log.Info("checking the head of channels...")
	headOfChannelPerPackage := mapHeadOfChannelsPerPackage(bundlesReport.Columns)
	mapPackagesWithMultiArchData := make(map[string][]MultipleArchitecturesBundleReport)

	for _, bundle := range headOfChannelPerPackage {

		// filter by the name
		if len(filter) > 0 {
			if !strings.Contains(bundle.PackageName, filter) {
				continue
			}
		}

		log.Infof("auditing for bundle %s", bundle.BundleCSV.Name)
		mb := MultipleArchitecturesBundleReport{BundleData: bundle}

		log.Infof("gathering data per bundle and performing the checks")
		manifestBundle := &manifests.Bundle{Name: mb.BundleData.PackageName, CSV: mb.BundleData.BundleCSV}
		multiArchValidator := validator.MultipleArchitecturesValidator.Validate(
			manifestBundle,
			map[string]string{"container-tools": containerTool},
		)
		if len(multiArchValidator) == 0 {
			log.Fatal("No objects could be validated")
		}

		log.Infof("processing the results to render the data in the report")
		mb.prepareDataPerBundle(multiArchValidator)

		// Add to the map the result per bundle to generate the report
		mapPackagesWithMultiArchData[mb.BundleData.PackageName] =
			append(mapPackagesWithMultiArchData[mb.BundleData.PackageName], mb)
	}

	multiArch.categorize(mapPackagesWithMultiArchData)
	multiArch.sort()
	return &multiArch
}

func (multiArch *MultipleArchitecturesReport) sort() {
	sort.Slice(multiArch.Unsupported[:], func(i, j int) bool {
		return multiArch.Unsupported[i].Name < multiArch.Unsupported[j].Name
	})

	sort.Slice(multiArch.Supported[:], func(i, j int) bool {
		return multiArch.Supported[i].Name < multiArch.Supported[j].Name
	})

	sort.Slice(multiArch.SupportedWithWarnings[:], func(i, j int) bool {
		return multiArch.SupportedWithWarnings[i].Name < multiArch.SupportedWithWarnings[j].Name
	})

	sort.Slice(multiArch.SupportedWithErrors[:], func(i, j int) bool {
		return multiArch.SupportedWithErrors[i].Name < multiArch.SupportedWithErrors[j].Name
	})
}

func (multiArch *MultipleArchitecturesReport) categorize(
	mapPackagesWithMultData map[string][]MultipleArchitecturesBundleReport) {
	for pkg, bundles := range mapPackagesWithMultData {
		//nolint: scopelint
		sort.Slice(bundles[:], func(i, j int) bool {
			return bundles[i].BundleData.BundleCSV.Name < bundles[j].BundleData.BundleCSV.Name
		})

		hasSupportOK := false
		hasSupportWarnings := false
		hasSupportErrors := false
		for _, bundle := range bundles {
			if bundle.HasMultiArchSupport && len(bundle.Errors) == 0 && len(bundle.Warnings) == 0 {
				hasSupportOK = true
			}
			if bundle.HasMultiArchSupport && len(bundle.Errors) == 0 && len(bundle.Warnings) > 0 {
				hasSupportWarnings = true
			}
			if bundle.HasMultiArchSupport && len(bundle.Errors) > 0 {
				hasSupportErrors = true
			}
		}

		if hasSupportWarnings {
			multiArch.SupportedWithWarnings = append(multiArch.SupportedWithWarnings,
				MultipleArchitecturesPackageReport{Name: pkg, Bundles: bundles})
		} else if hasSupportErrors {
			multiArch.SupportedWithErrors = append(multiArch.SupportedWithErrors,
				MultipleArchitecturesPackageReport{Name: pkg, Bundles: bundles})
		} else if hasSupportOK {
			multiArch.Supported = append(multiArch.Supported,
				MultipleArchitecturesPackageReport{Name: pkg, Bundles: bundles})
		} else {
			multiArch.Unsupported = append(multiArch.Unsupported,
				MultipleArchitecturesPackageReport{Name: pkg, Bundles: bundles})
		}
	}
}

// Build report data from CSV
func (mb *MultipleArchitecturesBundleReport) prepareDataPerBundle(multiArchValidator []errors.ManifestResult) {
	// Inspect CSV arch labels
	infraCSVArchLabels := []string{}
	for k, v := range mb.BundleData.BundleCSV.ObjectMeta.Labels {
		if strings.Contains(k, operatorFrameworkArchLabel) && v == "supported" {
			infraCSVArchLabels = append(infraCSVArchLabels, k)
		}
	}

	// Inspect CSV OS labels
	infraCSVOSLabels := []string{}
	for k, v := range mb.BundleData.BundleCSV.ObjectMeta.Labels {
		if strings.Contains(k, operatorFrameworkOSLabel) && v == "supported" {
			infraCSVOSLabels = append(infraCSVOSLabels, k)
		}
	}

	// Collect all of the labels together for the report
	mb.InfraLabelsUsed = append(mb.InfraLabelsUsed, infraCSVOSLabels...)
	mb.InfraLabelsUsed = append(mb.InfraLabelsUsed, infraCSVArchLabels...)

	// Gather images to be displayed in the report
	managerImages, allOtherImages := loadImagesFromCSV(*mb.BundleData.BundleCSV)

	// Look up any remaining platforms from CSV
	mb.AllArchFound = mb.gatherPlatformsFromCSV(infraCSVArchLabels, operatorFrameworkArchLabel, "amd64",
		func(platform platform) string { return platform.Architecture }, managerImages)
	mb.AllOsFound = mb.gatherPlatformsFromCSV(infraCSVOSLabels, operatorFrameworkOSLabel, "linux",
		func(platform platform) string { return platform.OS }, managerImages)

	mb.prepareImagesForReport(managerImages, allOtherImages)

	mb.checkIfHasMultiArch()
	for _, result := range multiArchValidator {
		for _, warning := range result.Warnings {
			mb.Warnings = append(mb.Warnings, warning.Error())
		}

		for _, err := range result.Errors {
			mb.Errors = append(mb.Errors, err.Error())
		}
	}

	// It is used to create the functions to show/hide the errors, warnings and images
	namehidden := mb.BundleData.BundleCSV.Name
	namehidden = strings.Replace(namehidden, "_", "", -1)
	namehidden = strings.Replace(namehidden, ".", "", -1)
	namehidden = strings.Replace(namehidden, "-", "", -1)
	mb.ForHideButton = namehidden
}

// prepareImagesForReport will ensure that
// manager Operator images(s) can be rendered in orange
// Images are format with the Platform found
func (mb *MultipleArchitecturesBundleReport) prepareImagesForReport(
	managerImages map[string][]platform, allOtherImages map[string][]platform) {
	for image, arrayData := range managerImages {
		allValues := []string{}
		for _, values := range arrayData {
			if len(values.OS) == 0 && len(values.Architecture) == 0 {
				continue
			}
			allValues = append(allValues, fmt.Sprintf("%s:%s", values.OS, values.Architecture))
		}
		mb.ManagerImage = append(mb.ManagerImage, fmt.Sprintf("(Operator Manager Image) %s:%q", image, allValues))
	}

	for image, arrayData := range allOtherImages {
		allValues := []string{}
		for _, values := range arrayData {
			if len(values.OS) == 0 && len(values.Architecture) == 0 {
				continue
			}
			allValues = append(allValues, fmt.Sprintf("%s:%s", values.OS, values.Architecture))
		}
		mb.Images = append(mb.Images, fmt.Sprintf("%s:%q", image, allValues))
	}
}

func (mb *MultipleArchitecturesBundleReport) checkIfHasMultiArch() {
	foundAnotherArch := false
	foundAnotherSO := false
	for k, v := range mb.AllArchFound {
		//nolint: goconst
		if v != "error" && k != "amd64" {
			foundAnotherArch = true
			break
		}
	}

	for k, v := range mb.AllOsFound {
		//nolint: goconst
		if v != "error" && k != "linux" {
			foundAnotherSO = true
		}
	}

	if foundAnotherArch || foundAnotherSO {
		mb.HasMultiArchSupport = true
	}
}

// operatorFrameworkArchLabel defines the label used to store the supported Arch on CSV
const operatorFrameworkArchLabel = "operatorframework.io/arch."

// operatorFrameworkOSLabel defines the label used to store the supported So on CSV
const operatorFrameworkOSLabel = "operatorframework.io/os."

// loadImagesFromCSV will add all allOtherImages found in the CSV
// it will be looking for all containers allOtherImages and what is defined
// via the spec.RELATE_IMAGE (required for disconnect support)
func loadImagesFromCSV(csv v1alpha1.ClusterServiceVersion) (map[string][]platform, map[string][]platform) {
	// We need to try looking for the manager image so that we can
	// be more assertive in the guess to warning the Operator
	// authors that they might are forgot to use add the labels
	// because we found images that provides more support
	var managerImages = make(map[string][]platform)
	for _, v := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		foundManager := false
		// For the default scaffold we have a container called manager
		for _, c := range v.Spec.Template.Spec.Containers {
			if c.Name == "manager" && len(managerImages[c.Image]) == 0 {
				managerImages[c.Image] = append(managerImages[c.Image], platform{})
				foundManager = true
				break
			}
		}
		// If we do not find a container called manager then we
		// will add all from the Deployment Specs which is not the
		// kube-rbac-proxy image scaffold by default
		if !foundManager {
			for _, c := range v.Spec.Template.Spec.Containers {
				if c.Name != "kube-rbac-proxy" && len(managerImages[c.Image]) == 0 {
					managerImages[c.Image] = append(managerImages[c.Image], platform{})
				}
			}
		}
	}

	var allOtherImages = make(map[string][]platform)
	if csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs != nil {
		for _, v := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
			for _, c := range v.Spec.Template.Spec.Containers {
				// If not be from manager the add
				if len(managerImages[c.Image]) == 0 {
					allOtherImages[c.Image] = append(allOtherImages[c.Image], platform{})
				}
			}
		}
	}

	for _, v := range csv.Spec.RelatedImages {
		allOtherImages[v.Image] = append(allOtherImages[v.Image], platform{})
	}

	return managerImages, allOtherImages
}

// extractValueFromOFArchLabel returns only the value for the desired label
// e.g. operatorframework.io/arch.amd64 ==> amd64
func extractValueFromOFLabel(v string, prefix string) string {
	label := strings.ReplaceAll(v, prefix, "")
	return label
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func mapHeadOfChannelsPerPackage(bundlesReport []bundles.Column) map[string]bundles.Column {
	mapPackagesWithBundles := make(map[string]bundles.Column)
	for _, v := range bundlesReport {
		if v.IsHeadOfChannel && !v.IsDeprecated && len(v.PackageName) > 0 && v.IsFromDefaultChannel {
			mapPackagesWithBundles[v.PackageName] = v
		}
	}
	return mapPackagesWithBundles
}

func (mb *MultipleArchitecturesBundleReport) gatherPlatformsFromCSV(
	infraCSVLabels []string,
	operatorFrameworkLabel string,
	defaultValue string,
	extractor func(platform) string,
	managerImages map[string][]platform) map[string]string {
	// Gather supported platforms
	platforms := map[string]string{}

	// Add the values provided via label
	for _, v := range infraCSVLabels {
		label := extractValueFromOFLabel(v, operatorFrameworkLabel)
		platforms[label] = label
	}

	// If a CSV does not include an arch label, it is treated as if it has the following AMD64 support label by default
	if len(infraCSVLabels) == 0 {
		platforms[defaultValue] = defaultValue
	}

	// Get all ARCH from the provided manager images
	for _, imageData := range managerImages {
		for _, platform := range imageData {
			if len(extractor(platform)) > 0 {
				mb.AllArchFound[extractor(platform)] = extractor(platform)
			}
		}
	}

	return platforms
}
