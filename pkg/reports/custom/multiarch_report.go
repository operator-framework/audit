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
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"

	"sort"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type MultipleArchitecturesBundle struct {
	BundleData          bundles.Column
	InfraLabelsUsed     []string
	AllArchFound        map[string]string
	AllOsFound          map[string]string
	Validations         []string
	Images              []string
	HasMultiArchSupport bool
	ForHideButton       string
}

type MultipleArchitecturesPackage struct {
	Name    string
	Bundles []MultipleArchitecturesBundle
}

type MultipleArchitecturesReport struct {
	ImageName           string
	ImageID             string
	ImageHash           string
	ImageBuild          string
	GeneratedAt         string
	Unsupported         []MultipleArchitecturesPackage
	Supported           []MultipleArchitecturesPackage
	SupportedWithErrors []MultipleArchitecturesPackage
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
	mapPerPkgHeadsOnly := mapHeadBundlesPerPackageWith(bundlesReport.Columns)
	mapPackagesWithMultData := make(map[string][]MultipleArchitecturesBundle)

	for pkg, bundles := range mapPerPkgHeadsOnly {
		for _, bundle := range bundles {
			// filter by the name
			if len(filter) > 0 {
				if !strings.Contains(bundle.PackageName, filter) {
					continue
				}
			}
			log.Infof("auditing for bundle %s", bundle.BundleCSV.Name)
			mb := MultipleArchitecturesBundle{BundleData: bundle}

			multiArchValidator := multiArchValidator{bundle: bundle.BundleCSV, containerTool: containerTool}
			multiArchValidator.validate()

			mb.AllArchFound = multiArchValidator.allArchFound
			mb.AllOsFound = multiArchValidator.allOsFound

			mb.InfraLabelsUsed = append(mb.InfraLabelsUsed, multiArchValidator.infraOSLabels...)
			mb.InfraLabelsUsed = append(mb.InfraLabelsUsed, multiArchValidator.infraArchLabels...)

			res := []string{}
			if multiArchValidator.images != nil {
				for image, arrayData := range multiArchValidator.images {
					allValues := []string{}
					for _, values := range arrayData {
						if len(values.OS) == 0 && len(values.Architecture) == 0 {
							continue
						}
						allValues = append(allValues, fmt.Sprintf("%s:%s", values.OS, values.Architecture))
					}
					res = append(res, fmt.Sprintf("%s:%q", image, allValues))
				}
			}

			mb.Images = res
			mb.checkIfHasMultiArch()

			for _, w := range multiArchValidator.warns {
				mb.Validations = append(mb.Validations, w.Error())
			}

			namehidden := bundle.BundleCSV.Name
			namehidden = strings.Replace(namehidden, "_", "", -1)
			namehidden = strings.Replace(namehidden, ".", "", -1)
			namehidden = strings.Replace(namehidden, "-", "", -1)
			mb.ForHideButton = namehidden

			mapPackagesWithMultData[pkg] = append(mapPackagesWithMultData[pkg], mb)
		}
	}

	for pkg, bundles := range mapPackagesWithMultData {

		//nolint: scopelint
		sort.Slice(bundles[:], func(i, j int) bool {
			return bundles[i].BundleData.BundleCSV.Name < bundles[j].BundleData.BundleCSV.Name
		})

		hasSupportOK := false
		hasSupportErrors := false
		for _, bundle := range bundles {
			if bundle.HasMultiArchSupport && len(bundle.Validations) == 0 {
				hasSupportOK = true
			}
			if bundle.HasMultiArchSupport && len(bundle.Validations) > 0 {
				hasSupportErrors = true
			}
		}

		if hasSupportErrors {
			multiArch.SupportedWithErrors = append(multiArch.SupportedWithErrors,
				MultipleArchitecturesPackage{Name: pkg, Bundles: bundles})
		} else if hasSupportOK {
			multiArch.Supported = append(multiArch.Supported,
				MultipleArchitecturesPackage{Name: pkg, Bundles: bundles})
		} else {
			multiArch.Unsupported = append(multiArch.Unsupported,
				MultipleArchitecturesPackage{Name: pkg, Bundles: bundles})
		}
	}

	sort.Slice(multiArch.Unsupported[:], func(i, j int) bool {
		return multiArch.Unsupported[i].Name < multiArch.Unsupported[j].Name
	})

	sort.Slice(multiArch.Supported[:], func(i, j int) bool {
		return multiArch.Supported[i].Name < multiArch.Supported[j].Name
	})

	sort.Slice(multiArch.SupportedWithErrors[:], func(i, j int) bool {
		return multiArch.SupportedWithErrors[i].Name < multiArch.SupportedWithErrors[j].Name
	})

	return &multiArch
}

func (mb *MultipleArchitecturesBundle) checkIfHasMultiArch() {
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

// multiArchValidator store the data to perform the tests
type multiArchValidator struct {
	// infraArchLabels store the arch labels (i.e amd64, ppc64le) from
	// operatorframework.io/arch.<GOARCH>: supported
	infraArchLabels []string
	// InfraSOLabel store the OS labels from
	// operatorframework.io/os.<GOARCH>: supported
	infraOSLabels []string
	// images stores the images defined in the bundle with the platform.arch supported
	images map[string][]platform
	// allArchFound contains a map of the arch types found
	allArchFound map[string]string
	// allOsFound contains a map of the so(s) found
	allOsFound map[string]string
	// Store the bundle load
	bundle *v1alpha1.ClusterServiceVersion
	// containerTool defines the container tool which will be used to inspect the images
	containerTool string
	// warns stores the errors faced by the validator to return the warnings
	warns []error
}

// manifestInspect store the data obtained by running container-tool manifest inspect <IMAGE>
type manifestInspect struct {
	ManifestData []manifestData `json:"manifests"`
}

// manifestData store the platforms
type manifestData struct {
	Platform platform `json:"platform"`
}

// platform store the Architecture and OS supported by the image
type platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// validate performs all required checks to validate the bundle against the Multiple Architecture
// configuration to guess the missing labels and/or highlight what are the missing Architectures
// for the images (for what is configured to be supported AND for what we guess that is supported
// and just is missing a label).
func (data *multiArchValidator) validate() {
	data.allArchFound = make(map[string]string)
	data.allOsFound = make(map[string]string)

	data.loadInfraLabelsFromCSV()
	data.loadImagesFromCSV()
	data.inspectAllImages()
	data.loadAllPossibleArchSupported()
	data.loadAllPossibleSoSupported()
	data.doChecks()
}

// loadInfraLabelsFromCSV will gather the respective labels from the CSV
func (data *multiArchValidator) loadInfraLabelsFromCSV() {
	for k, v := range data.bundle.ObjectMeta.Labels {
		if strings.Contains(k, operatorFrameworkArchLabel) && v == "supported" {
			data.infraArchLabels = append(data.infraArchLabels, k)
		}
	}
	for k, v := range data.bundle.ObjectMeta.Labels {
		if strings.Contains(k, operatorFrameworkOSLabel) && v == "supported" {
			data.infraOSLabels = append(data.infraOSLabels, k)
		}
	}
}

// loadImagesFromCSV will add all images found in the CSV
// it will be looking for all containers images and what is defined
// via the spec.RELATE_IMAGE (required for disconnect support)
func (data *multiArchValidator) loadImagesFromCSV() {
	data.images = make(map[string][]platform)
	if data.bundle.Spec.InstallStrategy.StrategySpec.DeploymentSpecs != nil {
		for _, v := range data.bundle.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
			for _, c := range v.Spec.Template.Spec.Containers {
				data.images[c.Image] = append(data.images[c.Image], platform{})
			}
		}
	}

	for _, v := range data.bundle.Spec.RelatedImages {
		data.images[v.Image] = append(data.images[v.Image], platform{})
	}
}

// runManifestInspect executes the command for we are able to check what
// are the Architecture(s) and SO(s) supported per each image found
func runManifestInspect(image, tool string) (manifestInspect, error) {
	log.Infof("running manifest inspect %s", image)
	cmd := exec.Command(tool, "manifest", "inspect", image)
	output, err := runCommand(cmd)
	if err != nil {
		return manifestInspect{}, err
	}

	var inspect manifestInspect
	if err := json.Unmarshal(output, &inspect); err != nil {
		return manifestInspect{}, err
	}
	return inspect, nil
}

// run executes the provided command within this context
func runCommand(cmd *exec.Cmd) ([]byte, error) {
	command := strings.Join(cmd.Args, " ")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}
	return output, nil
}

// inspectAllImages will perform the required steps to inspect all images found
func (data *multiArchValidator) inspectAllImages() {
	for k := range data.images {
		manifest, err := runManifestInspect(k, data.containerTool)
		if err != nil {
			// try once more
			manifest, err = runManifestInspect(k, data.containerTool)
			if err != nil {
				data.warns = append(data.warns, fmt.Errorf("unable to inspect the image (%s) : %s", k, err))

				// We set the Arch and SO as error for we are able to deal witth these cases further
				// Se that make no sense we raise a warning to notify the user that the image
				// does not provide some kind of support only because we were unable to inspect it.
				// Be aware that the validator raise warnings for all cases scenarios to let
				// the author knows that those were not checked at all and why.
				data.images[k] = append(data.images[k], platform{"error", "error"})
				continue
			}
		}

		if manifest.ManifestData != nil {
			for _, manifest := range manifest.ManifestData {
				data.images[k] = append(data.images[k], manifest.Platform)
			}
		}
	}

}

// doChecks centralize all checks which are done with this validator
func (data *multiArchValidator) doChecks() {
	data.checkMissingImagesSupport()
	data.checkSupportDefined()
	// Note that we can only check if the CSV is missing or not
	// label after check all possible arch/so supported
	// on the check above
	data.checkMissingLabelsForArchs()
	data.checkMissingLabelsForSO()
}

// checkMissingImagesSupport checks if any image is missing some arch or so found
// (probably the image should support the arch or SO )
func (data *multiArchValidator) checkMissingImagesSupport() {
	for image, plaformFromImage := range data.images {
		listArchNotFound := []string{}
		for archFromList := range data.allArchFound {
			found := false
			for _, imageData := range plaformFromImage {
				// Ignore the case when the Plataform.Architecture == "error" since that means
				// that was not possible to inspect the image
				if imageData.Architecture == "error" {
					found = true
					break
				}

				if imageData.Architecture == archFromList {
					found = true
					break
				}
			}
			if !found && archFromList != "error" {
				listArchNotFound = append(listArchNotFound, archFromList)
			}
		}
		if len(listArchNotFound) > 0 {
			sort.Strings(listArchNotFound)
			data.warns = append(data.warns,
				fmt.Errorf("check if the image (%s) should not support %q. "+
					"Note that this CSV has labels for this/those architecture(s) "+
					"OR one or more images defined on the CSV are providing this support."+
					"Then, it is very likely that you want to provide it", image, listArchNotFound))
		}

		listAllSoNotFound := []string{}
		for archOSList := range data.allOsFound {
			found := false
			for _, imageData := range plaformFromImage {
				// Ignore the case when the Plataform.Architecture == "error" since that means
				// that was not possible to inspect the image
				if imageData.OS == "error" {
					found = true
					break
				}

				if imageData.OS == archOSList {
					found = true
					break
				}
			}
			if !found && archOSList != "error" {
				listAllSoNotFound = append(listAllSoNotFound, archOSList)
			}
		}
		if len(listAllSoNotFound) > 0 {
			sort.Strings(listAllSoNotFound)
			data.warns = append(data.warns,
				fmt.Errorf("check if the image (%s) should not support %q. "+
					"Note that this CSV has labels for this/those SO(s) "+
					"OR one or more images defined on the CSV are providing this support."+
					"Then, it is very likely that you want to provide it",
					image, listAllSoNotFound))
		}
	}

}

// verify if 1 or more images has support for a SO not defined via the labels
// (probably the label for this SO is missing )
func (data *multiArchValidator) checkMissingLabelsForSO() {
	notFoundSoLabel := []string{}
	for supported := range data.allOsFound {
		found := false
		for _, infra := range data.infraOSLabels {
			if strings.Contains(infra, supported) {
				found = true
				break
			}
		}
		// If the value is linux and no labels were added to the CSV then it is fine
		if !found && supported != "error" {
			// if the only arch supported is linux then,  we should not ask for the label
			if !(supported == "linux" && len(data.allOsFound) == 1 && len(data.allOsFound["linux"]) > 0) {
				notFoundSoLabel = append(notFoundSoLabel, supported)
			}

		}
	}

	if len(notFoundSoLabel) > 0 {
		// We need to sort, otherwise it is possible verify in the tests that we have
		// this message as result
		sort.Strings(notFoundSoLabel)
		data.warns = append(data.warns,
			fmt.Errorf("check if the CSV is missing the label (%s<value>) for this/those SO(s): %q. "+
				"Note that this CSV has one or more images declared which provides this support. "+
				"Then, it is very likely that you want to provide it. "+
				"So that, if you support more than linux SO you MUST "+
				"use the required labels for all which are supported. "+
				"Otherwise, your solution might not filtered accordingly",
				operatorFrameworkOSLabel, notFoundSoLabel))
	}
}

// checkMissingLabelsForArchs verify if 1 or ore images has support for a Arch not defined via the labels
// (probably the label for this Arch is missing )
func (data *multiArchValidator) checkMissingLabelsForArchs() {
	notFoundArchLabel := []string{}
	for supported := range data.allArchFound {
		found := false
		for _, infra := range data.infraArchLabels {
			if strings.Contains(infra, supported) {
				found = true
				break
			}
		}
		// If the value is amd64 and no labels were added to the CSV then it is fine
		if !found && supported != "error" {
			// if the only arch supported is amd64 then we should not ask for the label
			if !(supported == "amd64" && len(data.allArchFound) == 1 && len(data.allArchFound["amd64"]) > 0) {
				notFoundArchLabel = append(notFoundArchLabel, supported)
			}
		}
	}

	if len(notFoundArchLabel) > 0 {
		// We need to sort, otherwise it is possible verify in the tests that we have
		// this message as result
		sort.Strings(notFoundArchLabel)
		data.warns = append(data.warns,
			fmt.Errorf("check if the CSV is missing the label (%s<value>) for this/those Arch(s): %q. "+
				"Note that this CSV has one or more images declared which provides this support. "+
				"Then, it is very likely that you want to provide it. "+
				"So that, if you support more than amd64 architectures you MUST "+
				"use the required labels for all which are supported. "+
				"Otherwise, your solution might not filtered accordingly",
				operatorFrameworkArchLabel, notFoundArchLabel))
	}
}

func (data *multiArchValidator) loadAllPossibleArchSupported() {
	// Add the values provided via label
	for _, v := range data.infraArchLabels {
		label := extractValueFromOFArchLabel(v)
		data.allArchFound[label] = label
	}

	// If a CSV does not include an arch label, it is treated as if it has the following AMD64 support label by default
	if len(data.infraArchLabels) == 0 {
		data.allArchFound["amd64"] = "amd64"
	}

	// Get all ARCH from the provided images
	for _, imageData := range data.images {
		for _, plataform := range imageData {
			if len(plataform.Architecture) > 0 {
				data.allArchFound[plataform.Architecture] = plataform.Architecture
			}
		}
	}
}

// loadAllPossibleSoSupported will verify all SO that this bundle can support
// for then, we aare able to check if it is missing labels.
// Note:
// - we check what are the SO of all images informed
// - we ensure that the linux SO will be added when none so labels were informed
// - we check all labels to know what are the SO(s) to obtain the list of them which the bundle is defining
func (data *multiArchValidator) loadAllPossibleSoSupported() {
	// Add the values provided via label
	for _, v := range data.infraOSLabels {
		label := extractValueFromOFSoLabel(v)
		data.allOsFound[label] = label
	}

	// If a ClusterServiceVersion does not include an os label, a target OS is assumed to be linux
	if len(data.infraOSLabels) == 0 {
		data.allOsFound["linux"] = "linux"
	}

	// Get all SO from the provided images
	for _, imageData := range data.images {
		for _, plataform := range imageData {
			if len(plataform.OS) > 0 {
				data.allOsFound[plataform.OS] = plataform.OS
			}
		}
	}
}

// checkSupportDefined checks if all images supports the ARCHs and SOs defined
func (data *multiArchValidator) checkSupportDefined() {
	configuredS0 := []string{}
	if len(data.infraOSLabels) == 0 {
		configuredS0 = []string{"linux"}
	}

	for _, label := range data.infraOSLabels {
		configuredS0 = append(configuredS0, extractValueFromOFSoLabel(label))
	}

	configuredArch := []string{}
	if len(data.infraArchLabels) == 0 {
		configuredArch = []string{"amd64"}
	}

	for _, label := range data.infraArchLabels {
		configuredArch = append(configuredArch, extractValueFromOFArchLabel(label))
	}

	allSupportedConfiguration := []string{}
	for _, so := range configuredS0 {
		for _, arch := range configuredArch {
			allSupportedConfiguration = append(allSupportedConfiguration, fmt.Sprintf("%s.%s", so, arch))
		}
	}

	notFoundImgPlat := map[string][]string{}
	for _, config := range allSupportedConfiguration {
		for image, allPlataformFromImage := range data.images {
			found := false
			for _, imgPlat := range allPlataformFromImage {
				// Ignore the errors since they mean that was not possible to inspect
				// the image
				if imgPlat.OS == "error" {
					found = true
					break
				}

				if config == fmt.Sprintf("%s.%s", imgPlat.OS, imgPlat.Architecture) {
					found = true
					break
				}
			}

			if !found {
				notFoundImgPlat[config] = append(notFoundImgPlat[config], image)
			}
		}
	}

	if len(notFoundImgPlat) > 0 {
		for platform, images := range notFoundImgPlat {
			// If we not sort the images we cannot check its result in the tests
			sort.Strings(images)
			data.warns = append(data.warns,
				fmt.Errorf("**ATTENTION**: The support for (SO.architecture): (%s) was not found for the image(s) %s. "+
					"However, this CSV is configured to provided these support",
					platform, images))
		}
	}
}

// extractValueFromOFSoLabel returns only the value of the SO label (i.e. linux)
func extractValueFromOFSoLabel(v string) string {
	label := strings.ReplaceAll(v, operatorFrameworkOSLabel, "")
	return label
}

// extractValueFromOFArchLabel returns only the value of the ARCH label (i.e. amd64)
func extractValueFromOFArchLabel(v string) string {
	label := strings.ReplaceAll(v, operatorFrameworkArchLabel, "")
	return label
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func mapHeadBundlesPerPackageWith(bundlesReport []bundles.Column) map[string][]bundles.Column {
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport {
		if v.IsHeadOfChannel && !v.IsDeprecated && len(v.PackageName) > 0 {
			mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
		}
	}
	return mapPackagesWithBundles
}
