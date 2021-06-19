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

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Column struct {
	PackageName                  string   `json:"packageName"`
	KindsDeprecateAPIs           []string `json:"kindsDeprecateAPIs,omitempty"`
	HasWebhooks                  bool     `json:"hasWebhooks,omitempty"`
	MultipleArchitectures        []string `json:"multipleArchitectures,omitempty"`
	HasValidatorErrors           bool     `json:"hasValidatorErrors,omitempty"`
	HasValidatorWarnings         bool     `json:"hasValidatorWarnings"`
	HasScorecardFailingTests     bool     `json:"hasScorecardFailingTests"`
	HasScorecardSuggestions      bool     `json:"hasScorecardSuggestions"`
	ValidatorErrors              []string `json:"validatorErrors,omitempty"`
	ValidatorWarnings            []string `json:"validatorWarnings,omitempty"`
	ScorecardErrors              []string `json:"scorecardErrors,omitempty"`
	ScorecardSuggestions         []string `json:"scorecardSuggestions,omitempty"`
	ScorecardFailingTests        []string `json:"scorecardFailingTests,omitempty"`
	HasInvalidSkipRange          bool     `json:"hasInvalidSkipRange,omitempty"`
	HasInvalidVersioning         bool     `json:"hasInvalidVersioning,omitempty"`
	IsMultiChannel               bool     `json:"isMultiChannel,omitempty"`
	HasSupportForAllNamespaces   bool     `json:"hasSupportForAllNamespaces,omitempty"`
	HasSupportForMultiNamespaces bool     `json:"hasSupportForMultiNamespaces,omitempty"`
	HasSupportForSingleNamespace bool     `json:"hasSupportForSingleNamespaces,omitempty"`
	HasSupportForOwnNamespaces   bool     `json:"hasSupportForOwnNamespaces,omitempty"`
	HasInfraAnnotation           bool     `json:"hasInfraAnnotation,omitempty"`
	HasPossiblePerformIssues     bool     `json:"hasPossiblePerformIssues,omitempty"`
	HasCustomScorecardTests      bool     `json:"hasCustomScorecardTests,omitempty"`
	AuditErrors                  []string `json:"errors,omitempty"`
}

func NewColumn(data *Data, auditPkg models.AuditPackage) *Column {
	col := Column{}
	col.PackageName = auditPkg.PackageName

	allBundles := getAllBundles(data.Flags.Label, auditPkg)

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
	foundSupportingAllNamespaces := false
	foundSupportingSingleNamespaces := false
	foundSupportingOwnNamespaces := false
	foundSupportingMultiNamespaces := false
	foundInfraSupport := false
	foundPossiblePerformIssues := false
	foundCustomScorecards := false
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

	return &col

}

func getAllBundles(label string, auditPkg models.AuditPackage) []bundles.Column {
	var allBundles []bundles.Column
	for _, v := range auditPkg.AuditBundle {
		// do not add bundle which has not the label
		if len(label) > 0 && !v.FoundLabel {
			continue
		}
		bundle := bundles.NewColumn(v)
		allBundles = append(allBundles, *bundle)
	}
	return allBundles
}
