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
	"sort"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type PartialComplying struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
}

type OK struct {
	Name            string
	Kinds           []string
	Bundles         []string
	Channels        []string
	BundlesMigrated []string
}

type NotComplying struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
}

type APIDashReport struct {
	ImageName        string
	ImageID          string
	ImageHash        string
	ImageBuild       string
	NotComplying     []NotComplying
	PartialComplying []PartialComplying
	GeneratedAt      string
	OK               []OK
}

// NewAPIDashReport returns the structure to render the Deprecate API custom dashboard
func NewAPIDashReport(bundlesReport bundles.Report) *APIDashReport {
	apiDash := APIDashReport{}
	apiDash.ImageName = bundlesReport.Flags.IndexImage
	apiDash.ImageID = bundlesReport.IndexImageInspect.ID
	apiDash.ImageBuild = bundlesReport.IndexImageInspect.Created
	apiDash.GeneratedAt = bundlesReport.GenerateAt

	mapPackagesWithBundles := MapBundlesPerPackage(bundlesReport)
	notComplying := MapPkgsNotComplyingWithDeprecateAPI122(mapPackagesWithBundles)
	complying := MapPkgsComplyingWithDeprecateAPI122(mapPackagesWithBundles)
	partialComplying := MapPkgsPartiallComplyingWithDeprecatedAPI122(mapPackagesWithBundles, complying, notComplying)

	for k, bundles := range complying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.OK = append(apiDash.OK, OK{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}

	for k, bundles := range notComplying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.NotComplying = append(apiDash.NotComplying, NotComplying{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}

	for k, bundles := range partialComplying {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.PartialComplying = append(apiDash.PartialComplying, PartialComplying{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
		})
	}
	return &apiDash

}

func getReportValues(bundles []bundles.Column) ([]string, []string, []string, []string) {
	var kinds []string
	var channels []string
	for _, b := range bundles {
		kinds = append(kinds, b.KindsDeprecateAPIs...)
	}
	for _, b := range bundles {
		channels = append(channels, b.Channels...)
	}
	var bundlesNotMigrated []string
	var bundlesMigrated []string
	for _, b := range bundles {
		if len(b.KindsDeprecateAPIs) > 0 {
			bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b))
		} else {
			bundlesMigrated = append(bundlesMigrated, buildBundleString(b))
		}
	}

	sort.Slice(bundlesNotMigrated[:], func(i, j int) bool {
		return bundlesNotMigrated[i] < bundlesNotMigrated[j]
	})

	sort.Slice(bundlesMigrated[:], func(i, j int) bool {
		return bundlesMigrated[i] < bundlesMigrated[j]
	})

	return kinds, channels, bundlesNotMigrated, bundlesMigrated
}

func buildBundleString(b bundles.Column) string {
	return fmt.Sprintf("%s - (label=%s,max=%s,channels=%s,head:%s)",
		b.BundleName,
		b.OCPLabel,
		GetMaxOCPValue(b),
		pkg.GetUniqueValues(b.Channels),
		pkg.GetYesOrNo(b.IsHeadOfChannel),
	)
}
