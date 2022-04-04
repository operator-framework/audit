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
	"strings"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

const OCPLabel = "com.redhat.openshift.versions"

type PotentialImpacted struct {
	Name    string
	Founds  []string
	Bundles []string
}

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
	ImageName         string
	ImageID           string
	ImageHash         string
	ImageBuild        string
	OCPVersion        string
	K8SVersion        string
	Migrated          []Migrated
	NotMigrated       []NotMigrated
	PotentialImpacted []PotentialImpacted
	GeneratedAt       string
}

const ocp413 = "4.13"
const k8s126 = "1.26"
const ocp412 = "4.12"
const k8s125 = "1.25"

// NewAPIDashReport returns the structure to render the Deprecate API custom dashboard
// nolint:dupl
func NewAPIDashReport(bundlesReport bundles.Report, optionalValues map[string]string, filterPkg string) *APIDashReport {
	apiDash := APIDashReport{}
	apiDash.ImageName = bundlesReport.Flags.IndexImage
	apiDash.ImageID = bundlesReport.IndexImageInspect.ID
	apiDash.ImageBuild = bundlesReport.IndexImageInspect.Created
	apiDash.GeneratedAt = bundlesReport.GenerateAt

	// k8sVersionKey defines the key which can be used by its consumers
	// to inform what is the K8S version that should be used to do the tests against.
	const k8sVersionKey = "k8s-version"

	k8sVersion := optionalValues[k8sVersionKey]

	switch k8sVersion {
	case k8s126:
		apiDash.OCPVersion = ocp413
		apiDash.K8SVersion = k8s126
	case k8s125:
		apiDash.OCPVersion = ocp412
		apiDash.K8SVersion = k8s125
	default:
		apiDash.OCPVersion = "4.9"
		apiDash.K8SVersion = "1.22"
	}

	var allBundles []BundleDeprecate
	for _, v := range bundlesReport.Columns {
		// filter by the name
		if len(filterPkg) > 0 {
			if !strings.Contains(v.PackageName, filterPkg) {
				continue
			}
		}
		bd := BundleDeprecate{BundleData: v}
		bd.AddDeprecateDataFromValidators()
		bd.AddPotentialWarning()
		allBundles = append(allBundles, bd)
	}

	mapPackagesWithBundles := MapBundlesPerPackage(allBundles)
	migrated := MapPkgsComplyingWithDeprecateAPI(mapPackagesWithBundles, apiDash.K8SVersion)
	notMigrated := make(map[string][]BundleDeprecate)
	for key := range mapPackagesWithBundles {
		if len(migrated[key]) == 0 {
			notMigrated[key] = mapPackagesWithBundles[key]
		}
	}

	for k, bundles := range migrated {
		kinds, channels, bundlesNotMigrated, bundlesMigrated := GetReportValues(bundles, apiDash.K8SVersion)
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
		kinds, channels, bundlesNotMigrated, bundlesMigrated := GetReportValues(bundles, apiDash.K8SVersion)
		apiDash.NotMigrated = append(apiDash.NotMigrated, NotMigrated{
			Name:            k,
			Kinds:           pkg.GetUniqueValues(kinds),
			Channels:        pkg.GetUniqueValues(channels),
			Bundles:         bundlesNotMigrated,
			BundlesMigrated: bundlesMigrated,
			AllBundles:      bundles,
		})
	}

	// Calculate the potential impacted by
	impactedPkgs := map[string]string{}

	//todo: we need clean up this code
	if apiDash.OCPVersion == ocp412 || apiDash.OCPVersion == ocp413 {
		for k, bundles := range mapPackagesWithBundles {
			var apis []string
			var foundBundles []string
			for _, b := range bundles {

				// Ignore the following cases
				if b.BundleData.BundleCSV == nil || len(b.BundleData.PackageName) == 0 || b.BundleData.IsDeprecated {
					continue
				}

				switch apiDash.OCPVersion {
				case ocp412:

					// Ignore when the max ocp version == these values
					if b.BundleData.MaxOCPVersion == "4.11" {
						continue
					}

					// Ignore if OCP label is < 4.12 or 4.13
					ocpLabel := b.BundleData.BundleImageLabels[OCPLabel]
					if len(ocpLabel) > 0 {
						if contains, _ := pkg.RangeContainsVersion(ocpLabel, ocp412, true); !contains {
							continue
						}
					}

					if len(b.Permissions1_25) == 0 {
						continue
					}

					apis = append(apis, b.Permissions1_25...)
					foundBundles = append(foundBundles, buildBundleStringPotential(b.BundleData, pkg.GetUniqueValues(apis)))
				case ocp413:

					// Ignore when the max ocp version == these values
					if b.BundleData.MaxOCPVersion == ocp412 {
						continue
					}

					// Ignore if OCP label is < 4.12 or 4.13
					ocpLabel := b.BundleData.BundleImageLabels[OCPLabel]
					if len(ocpLabel) > 0 {
						if contains, _ := pkg.RangeContainsVersion(ocpLabel, ocp413, true); !contains {
							continue
						}
					}

					if len(b.Permissions1_26) == 0 {
						continue
					}

					apis = append(apis, b.Permissions1_26...)
					foundBundles = append(foundBundles, buildBundleStringPotential(b.BundleData, pkg.GetUniqueValues(apis)))
				}

			}

			if len(foundBundles) > 0 {
				sort.Slice(foundBundles[:], func(i, j int) bool {
					return foundBundles[i] < foundBundles[j]
				})

				impactedPkgs[k] = "found"
				apiDash.PotentialImpacted = append(apiDash.PotentialImpacted, PotentialImpacted{
					Name:    k,
					Founds:  pkg.GetUniqueValues(apis),
					Bundles: foundBundles,
				})

			}
		}
	}

	return &apiDash

}

func GetReportValues(bundles []BundleDeprecate, k8sVersion string) ([]string, []string, []string, []string) {
	var msg []string
	var channels []string
	for _, b := range bundles {
		switch k8sVersion {
		case k8s126:
			msg = append(msg, b.ApisRemoved1_26...)
		case k8s125:
			msg = append(msg, b.ApisRemoved1_25...)
		default:
			msg = append(msg, b.ApisRemoved1_22...)
		}
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

		switch k8sVersion {
		case "1.26":
			if len(b.ApisRemoved1_26) > 0 {
				bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b.BundleData))
			} else {
				bundlesMigrated = append(bundlesMigrated, buildBundleString(b.BundleData))
			}
		case "1.25":
			if len(b.ApisRemoved1_25) > 0 {
				bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b.BundleData))
			} else {
				bundlesMigrated = append(bundlesMigrated, buildBundleString(b.BundleData))
			}
		default:
			if len(b.ApisRemoved1_22) > 0 {
				bundlesNotMigrated = append(bundlesNotMigrated, buildBundleString(b.BundleData))
			} else {
				bundlesMigrated = append(bundlesMigrated, buildBundleString(b.BundleData))
			}
		}
	}

	sort.Slice(bundlesNotMigrated[:], func(i, j int) bool {
		return bundlesNotMigrated[i] < bundlesNotMigrated[j]
	})

	sort.Slice(bundlesMigrated[:], func(i, j int) bool {
		return bundlesMigrated[i] < bundlesMigrated[j]
	})

	return msg, channels, bundlesNotMigrated, bundlesMigrated
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

func buildBundleStringPotential(b bundles.Column, pontential []string) string {
	return fmt.Sprintf("%s - (label=%s,max=%s,channels=%s,head:%s,defaultChannel:%s, deprecated:%s, RBAC: %q)",
		b.BundleCSV.Name,
		b.BundleImageLabels[OCPLabel],
		GetMaxOCPValue(b),
		pkg.GetUniqueValues(b.Channels),
		pkg.GetYesOrNo(b.IsHeadOfChannel),
		pkg.GetYesOrNo(b.IsFromDefaultChannel),
		pkg.GetYesOrNo(b.IsDeprecated),
		pontential,
	)
}
