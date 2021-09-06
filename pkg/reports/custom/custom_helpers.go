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

	"github.com/blang/semver"
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
func GetTheLatestBundleVersion(bundlesFromChannel []bundles.Column) string {
	var latestVersion string
	for _, v := range bundlesFromChannel {
		bundleVersionSemVer, _ := semver.ParseTolerant(v.BundleVersion)
		latestVersionSemVer, _ := semver.ParseTolerant(latestVersion)
		if bundleVersionSemVer.GT(latestVersionSemVer) {
			latestVersion = v.BundleVersion
		}
	}
	return latestVersion
}

// GetHeadOfChannelState returns the qtd. of head of channels which are OK and configured with max ocp version
func GetHeadOfChannelState(headOfChannels []bundles.Column) bool {
	for _, v := range headOfChannels {
		// In this case has a valid path
		if len(v.KindsDeprecateAPIs) == 0 && !v.IsDeprecated {
			return true
		}
	}
	return false
}

// BuildMapBundlesPerChannels returns a map of bundles per packages
func BuildMapBundlesPerChannels(bundlesPerPkg []bundles.Column) map[string][]bundles.Column {
	bundlesPerChannels := make(map[string][]bundles.Column)
	for _, b := range bundlesPerPkg {
		for _, c := range b.Channels {
			bundlesPerChannels[c] = append(bundlesPerChannels[c], b)
		}
	}
	return bundlesPerChannels
}

// MapBundlesPerPackage returns map with all bundles found per pkg name
func MapBundlesPerPackage(bundlesReport bundles.Report) map[string][]bundles.Column {
	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport.Columns {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}
	return mapPackagesWithBundles
}

func GetHeadOfChannels(bundlesOfPackage []bundles.Column) []bundles.Column {
	var headOfChannels []bundles.Column
	qtdHeads := 0
	for _, v := range bundlesOfPackage {
		if v.IsHeadOfChannel {
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
func GetLatestBundlesVersions(bundlesPerChannels map[string][]bundles.Column) []bundles.Column {
	var latestBundlesVersionsPerChannel []bundles.Column
	for _, bundlesFromChannel := range bundlesPerChannels {
		latest := GetTheLatestBundleVersion(bundlesFromChannel)
		for _, bd := range bundlesFromChannel {
			if bd.BundleVersion == latest {
				latestBundlesVersionsPerChannel = append(latestBundlesVersionsPerChannel, bd)
				continue
			}
		}
	}
	return latestBundlesVersionsPerChannel
}
