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

type Columns struct {
	PackageName                  string   `json:"packageName"`
	HasV1beta1CRD                string   `json:"hasV1beta1CRD,omitempty"`
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
	HasDependency                bool     `json:"hasGKVDependency,omitempty"`
	IsMultiChannel               bool     `json:"isMultiChannel,omitempty"`
	HasSupportForAllNamespaces   bool     `json:"hasSupportForAllNamespaces,omitempty"`
	HasSupportForMultiNamespaces bool     `json:"hasSupportForMultiNamespaces,omitempty"`
	HasSupportForSingleNamespace bool     `json:"hasSupportForSingleNamespaces,omitempty"`
	HasSupportForOwnNamespaces   bool     `json:"hasSupportForOwnNamespaces,omitempty"`
	HasInfraSupport              bool     `json:"hasInfraSupport,omitempty"`
	HasPossiblePerformIssues     bool     `json:"hasPossiblePerformIssues,omitempty"`
	CreationDates                []string `json:"creationDates,omitempty"`
	OCPLabel                     []string `json:"ocpLabel,omitempty"`
	AuditErrors                  []error  `json:"errors"`
}
