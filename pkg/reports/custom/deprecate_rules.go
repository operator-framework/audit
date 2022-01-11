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
	"strings"

	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type BundleDeprecate struct {
	BundleData        bundles.Column
	DeprecateAPIsMsgs []string
	ApisRemoved1_22   []string
	ApisRemoved1_25   []string
	ApisRemoved1_26   []string
}

// (Green) Complying
// If is not using deprecated API(s) at all in the head channels
// If has at least one channel head which is compatible with 4.9 (migrated)
// and the other head channels are with max ocp version
func MapPkgsComplyingWithDeprecateAPI122(
	mapPackagesWithBundles map[string][]BundleDeprecate) map[string][]BundleDeprecate {
	complying := make(map[string][]BundleDeprecate)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if !hasNotMigrated1_22(bundlesPerPkg) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if hasHeadOfChannelMigrated1_22(bundlesPerPkg) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}
	return complying
}

func hasNotMigrated1_22(bundlesPerPkg []BundleDeprecate) bool {
	foundNotMigrated := false
	for _, v := range bundlesPerPkg {
		if len(v.ApisRemoved1_22) > 0 && !v.BundleData.IsDeprecated {
			foundNotMigrated = true
			break
		}
	}
	return foundNotMigrated
}

func hasHeadOfChannelMigrated1_22(bundlesPerPkg []BundleDeprecate) bool {
	for _, v := range bundlesPerPkg {
		if (v.ApisRemoved1_22 == nil || len(v.ApisRemoved1_22) < 1) &&
			v.BundleData.IsHeadOfChannel && !v.BundleData.IsDeprecated {
			return true
		}
	}
	return false
}

func (bd *BundleDeprecate) AddDeprecateDataFromValidators() {
	for _, result := range bd.BundleData.ValidatorErrors {
		bd.setDeprecateMsg(result)
	}
	for _, result := range bd.BundleData.ValidatorWarnings {
		bd.setDeprecateMsg(result)
	}
}

func (bd *BundleDeprecate) setDeprecateMsg(result string) {
	if strings.Contains(result, "this bundle is using APIs which were deprecated") {
		bd.DeprecateAPIsMsgs = append(bd.DeprecateAPIsMsgs, result)
		if strings.Contains(result, "1.22") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_22, result)
		}
		if strings.Contains(result, "1.25") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_25, result)
		}
		if strings.Contains(result, "1.26") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_26, result)
		}
	}
}
