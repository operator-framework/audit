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

// (Green) Complying
// If is not using deprecated API(s) at all in the head channels
// If has at least one channel head which is compatible with 4.9 (migrated)
// and the other head channels are with max ocp version
func MapPkgsComplyingWithDeprecateAPI122(
	mapPackagesWithBundles map[string][]bundles.Column) map[string][]bundles.Column {
	complying := make(map[string][]bundles.Column)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if !hasNotMigrated(bundlesPerPkg) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if hasHeadOfChannelMigrated(bundlesPerPkg) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}
	return complying
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

func hasHeadOfChannelMigrated(bundlesPerPkg []bundles.Column) bool {
	for _, v := range bundlesPerPkg {
		if (v.KindsDeprecateAPIs == nil || len(v.KindsDeprecateAPIs) < 1) && v.IsHeadOfChannel {
			return true
		}
	}
	return false
}
