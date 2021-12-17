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

	"github.com/blang/semver/v4"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type OK struct {
	Name            string
	Kinds           []string
	Bundles         []string
	Channels        []string
	BundlesMigrated []string
	AllBundles      []BundleDeprecate
}

type NotOK struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
	AllBundles      []BundleDeprecate
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

	var allBundles []BundleDeprecate
	for _, v := range bundlesReport.Columns {
		bd := BundleDeprecate{BundleData: v}
		bd.AddDeprecateDataFromValidators()
		allBundles = append(allBundles, bd)
	}

	mapPackagesWithBundles := MapBundlesPerPackage(allBundles)
	isOK := mapPkgsComplyingMaxOcpVersion(mapPackagesWithBundles)
	isNotOK := make(map[string][]BundleDeprecate)
	for key := range mapPackagesWithBundles {
		if len(isOK[key]) == 0 {

			// Filter the bundles to output only what is not OK to make
			// easier the report conference
			var notOKBundles []BundleDeprecate
			for _, b := range mapPackagesWithBundles[key] {
				if !b.BundleData.IsDeprecated && len(b.ApisRemoved1_22) > 0 &&
					!isMaxOCPVersionLowerThan49(b.BundleData.MaxOCPVersion) {
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
	mapPackagesWithBundles map[string][]BundleDeprecate) map[string][]BundleDeprecate {
	complying := make(map[string][]BundleDeprecate)
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

func hasWrongMaxOcpVersion(bundlesPerPkg []BundleDeprecate) bool {
	for _, v := range bundlesPerPkg {
		if !v.BundleData.IsDeprecated && len(v.ApisRemoved1_22) > 0 &&
			!isMaxOCPVersionLowerThan49(v.BundleData.MaxOCPVersion) {
			return true
		}
	}
	return false
}

func isMaxOCPVersionLowerThan49(maxOCPVersion string) bool {
	if len(maxOCPVersion) == 0 {
		return false
	}

	maxOCPVersion = strings.ReplaceAll(maxOCPVersion, "\"", "")
	semVerVersionMaxOcp, err := semver.ParseTolerant(maxOCPVersion)
	if err != nil {
		return false
	}

	// OCP version where the apis v1beta1 is no longer supported
	const ocpVerV1beta1Unsupported = "4.9"
	semVerOCPV1beta1Unsupported, _ := semver.ParseTolerant(ocpVerV1beta1Unsupported)
	return !semVerVersionMaxOcp.GE(semVerOCPV1beta1Unsupported)
}
