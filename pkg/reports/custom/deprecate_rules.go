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
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

// (Amber) Partial complying
// if is not read or green then fail in the amber scenarios
func MapPkgsPartiallComplyingWithDeprecatedAPI122(mapPackagesWithBundles map[string][]bundles.Column,
	complying map[string][]bundles.Column, notComplying map[string][]bundles.Column) map[string][]bundles.Column {
	partialComplying := make(map[string][]bundles.Column)
	for key := range mapPackagesWithBundles {
		if !(len(complying[key]) > 0 || len(notComplying[key]) > 0) {
			partialComplying[key] = mapPackagesWithBundles[key]
		}
	}
	return partialComplying
}

// (Green) Complying
// If is not using deprecated API(s) at all in the head channels
// If has at least one channel head which is compatible with 4.9 (migrated)
// and the other head channels are with max ocp version
func MapPkgsComplyingWithDeprecateAPI122(
	mapPackagesWithBundles map[string][]bundles.Column) map[string][]bundles.Column {
	complying := make(map[string][]bundles.Column)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		headOfChannels := GetHeadOfChannels(bundlesPerPkg)
		foundOK, foundConfiguredAccordingly := GetHeadOfChannelState(headOfChannels)
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if !hasNotMigrated(bundlesPerPkg) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		qtdHeads := len(headOfChannels)
		if qtdHeads == foundOK || (foundOK > 0 && qtdHeads == foundOK+foundConfiguredAccordingly) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}
	return complying
}

// (Red) Not complying
// That are the packages which has none head channels compatible with 4.9 and/or configured accordingly
// with max ocp version set
func MapPkgsNotComplyingWithDeprecateAPI122(
	mapPackagesWithBundles map[string][]bundles.Column) map[string][]bundles.Column {
	notComplying := make(map[string][]bundles.Column)

	for key, bundlesPerPkg := range mapPackagesWithBundles {
		headOfChannels := GetHeadOfChannels(bundlesPerPkg)
		foundOK, foundConfiguredAccordingly := GetHeadOfChannelState(headOfChannels)
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if hasNotMigrated(bundlesPerPkg) {
				notComplying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if foundOK == 0 && foundConfiguredAccordingly == 0 {
			notComplying[key] = mapPackagesWithBundles[key]
		}
	}
	return notComplying
}

func hasNotMigrated(bundlesPerPkg []bundles.Column) bool {
	foundNotMigrated := false
	for _, v := range bundlesPerPkg {
		if len(v.KindsDeprecateAPIs) > 0 {
			foundNotMigrated = true
			break
		}
	}
	return foundNotMigrated
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
	if qtdHeads == 0 {
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
