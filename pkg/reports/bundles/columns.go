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

package bundles

import (
	"encoding/json"
	"fmt"

	"github.com/operator-framework/api/pkg/validation/errors"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
)

const olmproperties = "olm.properties"
const olmpdeprecate = "olm.deprecated"
const olmmaxOpenShiftVersion = "olm.maxOpenShiftVersion"

type Column struct {
	PackageName              string                          `json:"packageName"`
	BundleImagePath          string                          `json:"bundleImagePath,omitempty"`
	DefaultChannel           string                          `json:"defaultChannel,omitempty"`
	MaxOCPVersion            string                          `json:"maxOCPVersion,omitempty"`
	Channels                 []string                        `json:"bundleChannel,omitempty"`
	ValidatorErrors          []string                        `json:"validatorErrors,omitempty"`
	ValidatorWarnings        []string                        `json:"validatorWarnings,omitempty"`
	ScorecardErrors          []string                        `json:"scorecardErrors,omitempty"`
	ScorecardSuggestions     []string                        `json:"scorecardSuggestions,omitempty"`
	ScorecardFailingTests    []string                        `json:"scorecardFailingTests,omitempty"`
	AuditErrors              []string                        `json:"errors,omitempty"`
	HasPossiblePerformIssues bool                            `json:"hasPossiblePerformIssues"`
	HasCustomScorecardTests  bool                            `json:"hasCustomScorecardTests"`
	IsHeadOfChannel          bool                            `json:"isHeadOfChannel"`
	IsDeprecated             bool                            `json:"isDeprecated"`
	IsFromDefaultChannel     bool                            `json:"isFromDefaultChannel"`
	BundleImageLabels        map[string]string               `json:"bundleImageLabels,omitempty"`
	BundleAnnotations        map[string]string               `json:"bundleAnnotations,omitempty"`
	BundleCSV                *v1alpha1.ClusterServiceVersion `json:"csv,omitempty"`
	PropertiesFromDB         []pkg.PropertiesAnnotation      `json:"propertiesFromDB,omitempty"`
}

func NewColumn(v models.AuditBundle) *Column {
	col := Column{}
	col.PackageName = v.PackageName
	col.BundleImagePath = v.OperatorBundleImagePath
	col.DefaultChannel = v.DefaultChannel
	col.Channels = pkg.GetUniqueValues(v.Channels)
	col.AuditErrors = v.Errors
	col.HasCustomScorecardTests = v.HasCustomScorecardTests
	col.IsHeadOfChannel = v.IsHeadOfChannel
	col.BundleImageLabels = v.BundleImageLabels
	col.BundleAnnotations = v.BundleAnnotations
	col.PropertiesFromDB = v.PropertiesDB

	if v.Bundle != nil && v.Bundle.CSV != nil {
		col.BundleCSV = v.Bundle.CSV
	} else if v.CSVFromIndexDB != nil {
		col.BundleCSV = v.CSVFromIndexDB
	}

	col.AddDataFromScorecard(v.ScorecardResults)
	col.AddDataFromValidators(v.ValidatorsResults)
	col.SetMaxOpenshiftVersion()
	col.SetIsDeprecated()

	for _, i := range v.Channels {
		if i == v.DefaultChannel {
			col.IsFromDefaultChannel = true
			break
		}
	}

	return &col
}

func (c *Column) SetMaxOpenshiftVersion() {

	if c.BundleCSV != nil {
		cvsProperties := c.BundleCSV.Annotations[olmproperties]
		if len(cvsProperties) > 0 {
			var properList []pkg.PropertiesAnnotation
			err := json.Unmarshal([]byte(cvsProperties), &properList)
			if err != nil {
				c.AuditErrors = append(c.AuditErrors, fmt.Errorf("csv.Annotations has an invalid value specified "+
					"for %s", olmproperties).Error())
			} else {
				for _, v := range properList {
					if v.Type == olmmaxOpenShiftVersion {
						c.MaxOCPVersion = v.Value
						break
					}
				}
			}
		}

		if len(c.MaxOCPVersion) > 0 {
			return
		}

	}

	for _, v := range c.PropertiesFromDB {
		if v.Type == olmmaxOpenShiftVersion {
			c.MaxOCPVersion = v.Value
			break
		}
	}
}

func (c *Column) SetIsDeprecated() {

	if c.BundleCSV != nil {
		cvsProperties := c.BundleCSV.Annotations[olmproperties]
		if len(cvsProperties) > 0 {
			var properList []pkg.PropertiesAnnotation
			err := json.Unmarshal([]byte(cvsProperties), &properList)
			if err != nil {
				c.AuditErrors = append(c.AuditErrors, fmt.Errorf("csv.Annotations has an invalid value specified "+
					"for %s", olmproperties).Error())
			} else {
				for _, v := range properList {
					if v.Type == olmpdeprecate {
						c.IsDeprecated = true
						break
					}
				}
			}
		}

		if c.IsDeprecated {
			return
		}
	}

	for _, v := range c.PropertiesFromDB {
		if v.Type == olmpdeprecate {
			c.IsDeprecated = true
			break
		}
	}
}

func (c *Column) AddDataFromScorecard(scorecardResults v1alpha3.TestList) {
	for _, i := range scorecardResults.Items {
		for _, v := range i.Status.Results {
			c.ScorecardErrors = append(c.ScorecardErrors, v.Errors...)
			c.ScorecardSuggestions = append(c.ScorecardSuggestions, v.Suggestions...)
			if len(v.Errors) > 0 {
				c.ScorecardFailingTests = append(c.ScorecardFailingTests, v.Name)
			}
		}
	}
}

func (c *Column) AddDataFromValidators(results []errors.ManifestResult) {
	for _, i := range results {
		if i.HasError() {
			for _, e := range i.Errors {
				c.ValidatorErrors = append(c.ValidatorErrors, e.Error())
			}
		}
		if i.HasWarn() {
			for _, e := range i.Warnings {
				c.ValidatorWarnings = append(c.ValidatorWarnings, e.Error())
			}
		}
	}
}
