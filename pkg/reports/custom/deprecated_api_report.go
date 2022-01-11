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

const OCPLabel = "com.redhat.openshift.versions"

type Migrated struct {
	Name            string
	Kinds           []string
	Bundles         []string
	Channels        []string
	BundlesMigrated []string
	AllBundles      []BundleDeprecate
}

type NotMigrated struct {
	Name            string
	Kinds           []string
	Channels        []string
	Bundles         []string
	BundlesMigrated []string
	AllBundles      []BundleDeprecate
}

type APIDashReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	Migrated    []Migrated
	NotMigrated []NotMigrated
	GeneratedAt string
}

// NewAPIDashReport returns the structure to render the Deprecate API custom dashboard
// nolint:dupl
func NewAPIDashReport(bundlesReport bundles.Report) *APIDashReport {
	apiDash := APIDashReport{}
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
	migrated := MapPkgsComplyingWithDeprecateAPI122(mapPackagesWithBundles)
	notMigrated := make(map[string][]BundleDeprecate)
	for key := range mapPackagesWithBundles {
		if len(migrated[key]) == 0 {
			notMigrated[key] = mapPackagesWithBundles[key]
		}
	}

	for k, bundles := range migrated {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.Migrated = append(apiDash.Migrated, Migrated{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
			AllBundles:      bundles,
		})
	}

	for k, bundles := range notMigrated {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := getReportValues(bundles)
		apiDash.NotMigrated = append(apiDash.NotMigrated, NotMigrated{
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

func getReportValues(bundles []BundleDeprecate) ([]string, []string, []string, []string) {
	var mgs1_22 []string
	var channels []string
	for _, b := range bundles {
		mgs1_22 = append(mgs1_22, b.ApisRemoved1_22...)
	}
	for _, b := range bundles {
		channels = append(channels, b.BundleData.Channels...)
	}
	var bundlesNotMigrated []string
	var bundlesMigrated []string
	for _, b := range bundles {
		if b.BundleData.BundleCSV == nil || len(b.BundleData.PackageName) == 0 {
			continue
		}
		if len(b.ApisRemoved1_22) > 0 {
			bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b.BundleData))
		} else {
			bundlesMigrated = append(bundlesMigrated, buildBundleString(b.BundleData))
		}
	}

	sort.Slice(bundlesNotMigrated[:], func(i, j int) bool {
		return bundlesNotMigrated[i] < bundlesNotMigrated[j]
	})

	sort.Slice(bundlesMigrated[:], func(i, j int) bool {
		return bundlesMigrated[i] < bundlesMigrated[j]
	})

	return mgs1_22, channels, bundlesNotMigrated, bundlesMigrated
}

func buildBundleString(b bundles.Column) string {
	return fmt.Sprintf("%s - (label=%s,max=%s,channels=%s,head:%s,defaultChannel:%s, deprecated:%s)",
		b.BundleCSV.Name,
		b.BundleImageLabels[OCPLabel],
		GetMaxOCPValue(b),
		pkg.GetUniqueValues(b.Channels),
		pkg.GetYesOrNo(b.IsHeadOfChannel),
		pkg.GetYesOrNo(b.IsFromDefaultChannel),
		pkg.GetYesOrNo(b.IsDeprecated),
	)
}
