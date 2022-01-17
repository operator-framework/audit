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
	"sort"
	"strings"

	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type ValidatorReportBundle struct {
	BundleData  bundles.Column
	Validations []string
}

type ValidatorPkg struct {
	Name    string
	Bundles []ValidatorReportBundle
}

type ValidatorReport struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
	FilterBy    string
	Packages    []ValidatorPkg
}

// nolint:dupl
func NewValidatorReport(bundlesReport bundles.Report, filterPkg, filterValidator string) *ValidatorReport {
	validReport := ValidatorReport{}
	validReport.ImageName = bundlesReport.Flags.IndexImage
	validReport.ImageID = bundlesReport.IndexImageInspect.ID
	validReport.ImageBuild = bundlesReport.IndexImageInspect.Created
	validReport.GeneratedAt = bundlesReport.GenerateAt
	validReport.FilterBy = filterValidator

	mapPackagesWithBundles := make(map[string][]bundles.Column)
	for _, v := range bundlesReport.Columns {
		mapPackagesWithBundles[v.PackageName] = append(mapPackagesWithBundles[v.PackageName], v)
	}

	mapPackagesWithValidations := make(map[string][]ValidatorReportBundle)

	for pkg, bundles := range mapPackagesWithBundles {

		if len(pkg) == 0 {
			continue
		}

		for _, bundle := range bundles {
			// filter by the name
			if len(filterPkg) > 0 {
				if !strings.Contains(bundle.PackageName, filterPkg) {
					continue
				}
			}

			if bundle.IsDeprecated {
				continue
			}

			mb := ValidatorReportBundle{BundleData: bundle}

			for _, vw := range bundle.ValidatorWarnings {
				if strings.Contains(vw, filterValidator) {
					mb.Validations = append(mb.Validations, vw)
				}
			}

			for _, vw := range bundle.ValidatorErrors {
				if strings.Contains(vw, filterValidator) {
					mb.Validations = append(mb.Validations, vw)
				}
			}

			if len(mb.Validations) > 0 {
				mapPackagesWithValidations[pkg] = append(mapPackagesWithValidations[pkg], mb)
			}
		}
	}

	for pkg, bundles := range mapPackagesWithValidations {
		validReport.Packages = append(validReport.Packages, ValidatorPkg{Name: pkg, Bundles: bundles})
	}

	sort.Slice(validReport.Packages[:], func(i, j int) bool {
		return validReport.Packages[i].Name < validReport.Packages[j].Name
	})

	return &validReport
}
