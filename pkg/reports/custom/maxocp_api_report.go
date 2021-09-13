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
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type OK struct {
	Name            string
	Kinds           []string
	Bundles         []string
	Channels        []string
	BundlesMigrated []string
	AllBundles      []bundles.Column
}

type NotOK struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
	AllBundles      []bundles.Column
}

type MaxDashReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	OK          []OK
	NotOK       []NotOK
	GeneratedAt string
}

// NewAPIDashReport returns the structure to render the Deprecate API custom dashboard
// nolint:dupl
func NewMaxDashReport(bundlesReport bundles.Report) *MaxDashReport {
	apiDash := MaxDashReport{}
	apiDash.ImageName = bundlesReport.Flags.IndexImage
	apiDash.ImageID = bundlesReport.IndexImageInspect.ID
	apiDash.ImageBuild = bundlesReport.IndexImageInspect.Created
	apiDash.GeneratedAt = bundlesReport.GenerateAt

	mapPackagesWithBundles := MapBundlesPerPackage(bundlesReport)
	isOK := mapPkgsComplyingMaxOcpVersion(mapPackagesWithBundles)
	isNotOK := make(map[string][]bundles.Column)
	for key := range mapPackagesWithBundles {
		if len(isOK[key]) == 0 {

			// Filter the bundles to output only what is not OK to make
			// easier the report conference
			var notOKBundles []bundles.Column
			for _, b := range mapPackagesWithBundles[key] {
				if b.KindsDeprecateAPIs != nil &&
					b.KindsDeprecateAPIs[0] != pkg.Unknown &&
					len(b.KindsDeprecateAPIs) > 0 && !pkg.IsMaxOCPVersionLowerThan49(b.MaxOCPVersion) {
					notOKBundles = append(notOKBundles, b)
				}
			}
			isNotOK[key] = notOKBundles
		}
	}

	for k, bundles := range isNotOK {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.NotOK = append(apiDash.NotOK, NotOK{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
			AllBundles:      bundles,
		})
	}

	return &apiDash

}

// (Green) Complying
// Return all pkgs that has all bundles using the removed APIs set with max ocp version
func mapPkgsComplyingMaxOcpVersion(
	mapPackagesWithBundles map[string][]bundles.Column) map[string][]bundles.Column {
	complying := make(map[string][]bundles.Column)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, we will ignore this cases
		if key == "" {
			continue
		}

		if !hasWrongMaxOcpVersion(bundlesPerPkg) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}
	return complying
}

func hasWrongMaxOcpVersion(bundlesPerPkg []bundles.Column) bool {
	for _, v := range bundlesPerPkg {
		if v.KindsDeprecateAPIs != nil &&
			v.KindsDeprecateAPIs[0] != pkg.Unknown &&
			len(v.KindsDeprecateAPIs) > 0 &&
			!pkg.IsMaxOCPVersionLowerThan49(v.MaxOCPVersion) {
			return true
		}
	}
	return false
}
