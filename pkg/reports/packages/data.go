// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packages

import (
	"fmt"
	"log"
	"sort"

	"github.com/blang/semver"

	sq "github.com/Masterminds/squirrel"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Data struct {
	AuditPackage             []models.AuditPackage
	HeadOperatorBundleReport []bundles.Columns
	Flags                    BindFlags
	IndexImageInspect        pkg.DockerInspectManifest
}

func (d *Data) PrepareReport() Report {
	var allColumns []Columns
	for _, auditPkg := range d.AuditPackage {

		col := Columns{}
		col.PackageName = auditPkg.PackageName

		allBundles := d.getAllBundles(auditPkg)

		var auditErrors []string
		var validatorErrors []string
		var validatorWarnings []string
		var scorecardErrors []string
		var scorecardSuggestions []string
		var scorecardFailingTests []string
		var muiltArchSupport []string
		var kindsFromRemovedAPI []string

		foundWebhooks := false
		foundScorecardSuggestions := false
		foundScorecardFailingTests := false
		foundValidatorWarnings := false
		foundValidatorErrors := false
		foundInvalidSkipRange := false
		foundInvalidVersioning := false
		foundDependency := false
		foundSupportingAllNamespaces := false
		foundSupportingSingleNamespaces := false
		foundSupportingOwnNamespaces := false
		foundSupportingMultiNamespaces := false
		foundInfraSupport := false
		foundPossiblePerformIssues := false
		foundCustomScorecards := false
		foundBundlesNotFollowingSuggestionsForDeprecatedAPIs := false
		qtUnknown := 0
		var uniqueChannelsFound []string

		for _, v := range allBundles {
			auditErrors = append(auditErrors, v.AuditErrors...)
			validatorErrors = append(validatorErrors, v.ValidatorErrors...)
			validatorWarnings = append(validatorWarnings, v.ValidatorWarnings...)
			scorecardErrors = append(scorecardErrors, v.ScorecardErrors...)
			scorecardSuggestions = append(scorecardSuggestions, v.ScorecardSuggestions...)
			scorecardFailingTests = append(scorecardFailingTests, v.ScorecardFailingTests...)
			muiltArchSupport = append(muiltArchSupport, v.MultipleArchitectures...)
			kindsFromRemovedAPI = append(kindsFromRemovedAPI, v.KindsDeprecateAPIs...)
			if len(v.KindsDeprecateAPIs) > 0 && v.KindsDeprecateAPIs[0] == pkg.Unknown {
				qtUnknown++
			}
			uniqueChannelsFound = append(uniqueChannelsFound, v.Channels...)

			if !foundScorecardSuggestions {
				foundScorecardSuggestions = len(v.ScorecardSuggestions) > 0
			}
			if !foundScorecardFailingTests {
				foundScorecardFailingTests = len(v.ScorecardFailingTests) > 0
			}
			if !foundValidatorWarnings {
				foundValidatorWarnings = len(v.ValidatorWarnings) > 0
			}
			if !foundValidatorErrors {
				foundValidatorErrors = len(v.ValidatorErrors) > 0
			}
			if !foundWebhooks && v.HasWebhook {
				foundWebhooks = true
			}
			if !foundInvalidVersioning && v.InvalidVersioning == pkg.GetYesOrNo(true) {
				foundInvalidVersioning = true
			}
			if !foundInvalidSkipRange && len(v.InvalidSkipRange) > 0 && v.InvalidSkipRange == pkg.GetYesOrNo(true) {
				foundInvalidSkipRange = true
			}
			if !foundDependency {
				foundDependency = v.HasDependency
			}
			if !foundSupportingAllNamespaces {
				foundSupportingAllNamespaces = v.IsSupportingAllNamespaces
			}
			if !foundSupportingOwnNamespaces {
				foundSupportingOwnNamespaces = v.IsSupportingOwnNamespaces
			}
			if !foundSupportingMultiNamespaces {
				foundSupportingMultiNamespaces = v.IsSupportingMultiNamespaces
			}
			if !foundSupportingSingleNamespaces {
				foundSupportingSingleNamespaces = v.IsSupportingSingleNamespace
			}
			if !foundInfraSupport {
				foundInfraSupport = len(v.Infrastructure) > 0
			}
			if !foundPossiblePerformIssues {
				foundPossiblePerformIssues = v.HasPossiblePerformIssues
			}
			if !foundCustomScorecards {
				foundCustomScorecards = v.HasCustomScorecardTests
			}
			if !foundBundlesNotFollowingSuggestionsForDeprecatedAPIs {
				if v.IsDeprecationAPIsSuggestionsSet == pkg.GetYesOrNo(false) {
					foundBundlesNotFollowingSuggestionsForDeprecatedAPIs = true
				}
			}
		}

		uniqueChannelsFound = pkg.GetUniqueValues(uniqueChannelsFound)
		col.IsMultiChannel = len(uniqueChannelsFound) > 0
		col.AuditErrors = auditErrors
		col.ScorecardFailingTests = scorecardFailingTests
		col.ScorecardSuggestions = scorecardSuggestions
		col.ValidatorWarnings = validatorWarnings
		col.ScorecardErrors = scorecardErrors
		col.ValidatorErrors = validatorErrors
		col.MultipleArchitectures = muiltArchSupport
		col.HasScorecardFailingTests = foundScorecardFailingTests
		col.HasScorecardSuggestions = foundScorecardSuggestions
		col.HasValidatorWarnings = foundValidatorWarnings
		col.HasValidatorErrors = foundValidatorErrors
		col.HasInvalidSkipRange = foundInvalidSkipRange
		col.HasInvalidVersioning = foundInvalidVersioning
		col.HasSupportForAllNamespaces = foundSupportingAllNamespaces
		col.HasSupportForMultiNamespaces = foundSupportingMultiNamespaces
		col.HasSupportForOwnNamespaces = foundSupportingOwnNamespaces
		col.HasSupportForSingleNamespace = foundSupportingSingleNamespaces
		col.HasInfraAnnotation = foundInfraSupport
		col.HasPossiblePerformIssues = foundPossiblePerformIssues
		col.KindsDeprecateAPIs = pkg.GetUniqueValues(kindsFromRemovedAPI)
		col.HasCustomScorecardTests = foundCustomScorecards

		// If was not possible get any bundle then needs to be Unknown
		if qtUnknown > 0 {
			if len(allBundles) == qtUnknown {
				col.KindsDeprecateAPIs[0] = pkg.Unknown
			}
			col.AuditErrors = append(col.AuditErrors,
				fmt.Errorf("unable to check the "+
					"removed API(s) for %d of %d head bundles of this package",
					qtUnknown, len(allBundles)).Error())
		}

		if len(col.KindsDeprecateAPIs) == 0 {
			col.HasDeprecateAPIsSuggestionsSet = pkg.NotRequired
		} else {
			col.HasDeprecateAPIsSuggestionsSet = pkg.GetYesOrNo(!foundBundlesNotFollowingSuggestionsForDeprecatedAPIs)
		}

		allColumns = append(allColumns, col)
	}

	sort.Slice(allColumns[:], func(i, j int) bool {
		return allColumns[i].PackageName < allColumns[j].PackageName
	})

	finalReport := Report{}
	finalReport.Flags = d.Flags
	finalReport.Columns = allColumns
	finalReport.IndexImageInspect = d.IndexImageInspect

	if len(allColumns) == 0 {
		log.Fatal("No data was found for the criteria informed. " +
			"Please, ensure that you provide valid information.")
	}

	return finalReport
}

func (d *Data) getAllBundles(auditPkg models.AuditPackage) []bundles.Columns {
	var allBundles []bundles.Columns

	// todo: check how to cleanup and centralize the bundle process
	// in the bundle for not duplicate the code and avoid issues

	for _, v := range auditPkg.AuditBundle {
		// do not add bundle which has not the label
		if len(d.Flags.Label) > 0 && !v.FoundLabel {
			continue
		}

		bundles := bundles.Columns{}

		var csv *v1alpha1.ClusterServiceVersion
		if v.Bundle != nil && v.Bundle.CSV != nil {
			csv = v.Bundle.CSV
		} else if v.CSVFromIndexDB != nil {
			csv = v.CSVFromIndexDB
		}

		bundles.AddDataFromCSV(csv)
		bundles.AddDataFromBundle(v.Bundle)
		bundles.AddDataFromScorecard(v.ScorecardResults)
		bundles.AddDataFromValidators(v.ValidatorsResults)
		bundles.SetMaxOpenshiftVersion(csv, v.PropertiesDB)

		bundles.BundleImageBuildDate = v.BuildAt
		bundles.OCPLabel = v.OCPLabel
		bundles.HasCustomScorecardTests = v.HasCustomScorecardTests

		if len(bundles.BundleVersion) < 1 && len(v.VersionDB) > 0 {
			bundles.BundleVersion = v.VersionDB
		}

		if len(bundles.BundleVersion) > 0 {
			_, err := semver.Parse(bundles.BundleVersion)
			if err != nil {
				bundles.InvalidVersioning = pkg.GetYesOrNo(true)
			} else {
				bundles.InvalidVersioning = pkg.GetYesOrNo(false)
			}
		}

		if len(bundles.SkipRange) > 0 {
			_, err := semver.ParseRange(bundles.SkipRange)
			if err != nil {
				bundles.InvalidSkipRange = pkg.GetYesOrNo(true)
			} else {
				bundles.InvalidSkipRange = pkg.GetYesOrNo(false)
			}
		}

		// Check if the bundle comply with the deprecated criteria
		if len(bundles.KindsDeprecateAPIs) > 0 {
			bundles.IsDeprecationAPIsSuggestionsSet = pkg.GetYesOrNo(
				pkg.IsComplyingWithDeprecatedCriteria(bundles.MaxOCPVersion, bundles.OCPLabel))
		} else {
			bundles.IsDeprecationAPIsSuggestionsSet = pkg.NotRequired
		}

		allBundles = append(allBundles, bundles)
	}
	return allBundles
}

func (d *Data) OutputReport() error {
	report := d.PrepareReport()
	switch d.Flags.OutputFormat {
	case pkg.Xls:
		if err := report.writeXls(); err != nil {
			return err
		}
	case pkg.JSON:
		if err := report.writeJSON(); err != nil {
			return err
		}
	case pkg.All:
		if err := report.writeXls(); err != nil {
			return err
		}
		if err := report.writeJSON(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format : %s", d.Flags.OutputFormat)
	}
	return nil
}

func (d *Data) BuildPackagesQuery() (string, error) {
	query := sq.Select("name, default_channel").From("package")

	if d.Flags.Limit > 0 {
		query = query.Limit(uint64(d.Flags.Limit))
	}

	if len(d.Flags.Filter) > 0 {
		like := "'%" + d.Flags.Filter + "%'"
		query = query.Where(fmt.Sprintf("name like %s", like))
	}

	query.OrderBy("name")
	sql, _, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("unable to create sql : %s", err)
	}
	return sql, nil
}
