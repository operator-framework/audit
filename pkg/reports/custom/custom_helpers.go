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
	"strings"

	"github.com/blang/semver/v4"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

// ParseBundlesJSONReport parse the JSON result from the audit-tool index bundle report and return its structure
func ParseBundlesJSONReport() (bundles.Report, error) {
	byteValue, err := pkg.ReadFile(Flags.File)
	if err != nil {
		return bundles.Report{}, err
	}
	var bundlesReport bundles.Report
	if err = json.Unmarshal(byteValue, &bundlesReport); err != nil {
		return bundles.Report{}, err
	}
	return bundlesReport, err
}

// ParseBundlesJSONReport parse the JSON result from the audit-tool index bundle report and return its structure
func ParseMultiBundlesJSONReport() ([]bundles.Report, error) {

	allFiles := strings.Split(Flags.Files, ";")
	var all []bundles.Report
	for _, file := range allFiles {
		if len(file) == 0 {
			continue
		}
		byteValue, err := pkg.ReadFile(file)
		if err != nil {
			return all, err
		}
		var bundlesReport bundles.Report
		if err = json.Unmarshal(byteValue, &bundlesReport); err != nil {
			return all, err
		}
		all = append(all, bundlesReport)
	}

	return all, nil
}

// GetMaxOCPValue returns the Max OCP annotation find on the bundle or an string not set to define
// that it was not set
func GetMaxOCPValue(b bundles.Column) string {
	maxValue := b.MaxOCPVersion
	if len(maxValue) == 0 {
		maxValue = "not set"
	}
	return maxValue
}

// GetTheLatestBundleVersion returns the latest/upper semversion
func GetTheLatestBundleVersion(bundlesFromChannel []BundleDeprecate) string {
	latestVersion, _ := semver.ParseTolerant("0.0.0")
	for _, v := range bundlesFromChannel {
		if v.BundleData.BundleCSV.Spec.Version.Version.GT(latestVersion) {
			latestVersion = v.BundleData.BundleCSV.Spec.Version.Version
		}
	}
	return latestVersion.String()
}

// BuildMapBundlesPerChannels returns a map of bundles per packages
func BuildMapBundlesPerChannels(bundlesPerPkg []BundleDeprecate) map[string][]BundleDeprecate {
	bundlesPerChannels := make(map[string][]BundleDeprecate)
	for _, b := range bundlesPerPkg {
		for _, c := range b.BundleData.Channels {
			bundlesPerChannels[c] = append(bundlesPerChannels[c], b)
		}
	}
	return bundlesPerChannels
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func MapBundlesPerPackage(bundlesReport []BundleDeprecate) map[string][]BundleDeprecate {
	mapPackagesWithBundles := make(map[string][]BundleDeprecate)
	for _, v := range bundlesReport {
		mapPackagesWithBundles[v.BundleData.PackageName] = append(mapPackagesWithBundles[v.BundleData.PackageName], v)
	}
	return mapPackagesWithBundles
}

func GetHeadOfChannels(bundlesOfPackage []BundleDeprecate) []BundleDeprecate {
	var headOfChannels []BundleDeprecate
	qtdHeads := 0
	for _, v := range bundlesOfPackage {
		if v.BundleData.IsHeadOfChannel {
			qtdHeads++
			headOfChannels = append(headOfChannels, v)
		}
	}

	bundlesPerChannels := BuildMapBundlesPerChannels(bundlesOfPackage)

	// If for the package has no bundle set in the channels
	// table as head of the channel then, we need to check
	// the scenarios
	if qtdHeads == 0 || qtdHeads != len(bundlesPerChannels) {
		headOfChannels = GetLatestBundlesVersions(bundlesPerChannels)
	}
	return headOfChannels
}

// GetQtLatestVersionChannelsState returns the qtd. of channels which are OK and configured with max ocp version
func GetLatestBundlesVersions(bundlesPerChannels map[string][]BundleDeprecate) []BundleDeprecate {
	var latestBundlesVersionsPerChannel []BundleDeprecate
	for _, bundlesFromChannel := range bundlesPerChannels {
		latest := GetTheLatestBundleVersion(bundlesFromChannel)
		for _, bd := range bundlesFromChannel {
			if bd.BundleData.BundleCSV.Spec.Version.String() == latest {
				latestBundlesVersionsPerChannel = append(latestBundlesVersionsPerChannel, bd)
				continue
			}
		}
	}
	return latestBundlesVersionsPerChannel
}
