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
	"strings"

	"github.com/blang/semver"
	"github.com/operator-framework/audit/pkg/models"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	validationerrors "github.com/operator-framework/api/pkg/validation/errors"

	"github.com/operator-framework/audit/pkg"
)

const certifiedAnnotation = "certified"
const repositoryAnnotation = "repository"
const archLabels = "operatorframework.io/arch."
const osLabel = "operatorframework.io/os."
const sdkBuilderAnnotation = "operators.operatorframework.io/builder"
const skipRangeAnnotation = "olm.skipRange"
const sdkProjectLayoutAnnotation = "operators.operatorframework.io/project_layout"
const infrastructureAnnotation = "operators.openshift.io/infrastructure-features"
const olmproperties = "olm.properties"
const olmmaxOpenShiftVersion = "olm.maxOpenShiftVersion"

type Column struct {
	PackageName                 string              `json:"packageName"`
	BundleName                  string              `json:"bundleName"`
	BundleVersion               string              `json:"bundleVersion,omitempty"`
	BundleImagePath             string              `json:"bundleImagePath,omitempty"`
	BundleImageBuildDate        string              `json:"bundleImageBuildDate,omitempty"`
	Repository                  string              `json:"repository,omitempty"`
	DefaultChannel              string              `json:"defaultChannel,omitempty"`
	Maturity                    string              `json:"maturity,omitempty"`
	Capabilities                string              `json:"capabilities,omitempty"`
	Categories                  string              `json:"categories,omitempty"`
	Builder                     string              `json:"builder,omitempty"`
	SDKVersion                  string              `json:"sdkVersion,omitempty"`
	ProjectLayout               string              `json:"projectLayout,omitempty"`
	InvalidVersioning           string              `json:"invalidVersioning,omitempty"`
	InvalidSkipRange            string              `json:"invalidSkipRange,omitempty"`
	SkipRange                   string              `json:"skipRange,omitempty"`
	Replace                     string              `json:"replace,omitempty"`
	Infrastructure              string              `json:"infrastructure,omitempty"`
	OCPLabel                    string              `json:"ocpLabel,omitempty"`
	MaxOCPVersion               string              `json:"maxOCPVersion,omitempty"`
	KindsDeprecateAPIs          []string            `json:"kindsDeprecateAPIs,omitempty"`
	Channels                    []string            `json:"bundleChannel,omitempty"`
	MultipleArchitectures       []string            `json:"multipleArchitectures,omitempty"`
	ValidatorErrors             []string            `json:"validatorErrors,omitempty"`
	ValidatorWarnings           []string            `json:"validatorWarnings,omitempty"`
	ScorecardErrors             []string            `json:"scorecardErrors,omitempty"`
	ScorecardSuggestions        []string            `json:"scorecardSuggestions,omitempty"`
	ScorecardFailingTests       []string            `json:"scorecardFailingTests,omitempty"`
	AuditErrors                 []string            `json:"errors,omitempty"`
	Skips                       []string            `json:"skips,omitempty"`
	DeprecateAPIsManifests      map[string][]string `json:"deprecateAPIsManifests,omitempty"`
	MaintainersEmail            []string            `json:"maintainersEmail,omitempty"`
	Links                       []string            `json:"links,omitempty"`
	Certified                   bool                `json:"certified"`
	HasWebhook                  bool                `json:"hasWebhook"`
	IsSupportingAllNamespaces   bool                `json:"supportsAllNamespaces"`
	IsSupportingMultiNamespaces bool                `json:"supportsMultiNamespaces"`
	IsSupportingSingleNamespace bool                `json:"supportSingleNamespaces"`
	IsSupportingOwnNamespaces   bool                `json:"supportsOwnNamespaces"`
	HasPossiblePerformIssues    bool                `json:"hasPossiblePerformIssues"`
	HasCustomScorecardTests     bool                `json:"hasCustomScorecardTests"`
	IsHeadOfChannel             bool                `json:"isHeadOfChannel"`
}

func NewColumn(v models.AuditBundle) *Column {
	col := Column{}
	col.InvalidSkipRange = pkg.NotUsed
	col.InvalidVersioning = pkg.Unknown
	col.PackageName = v.PackageName
	col.BundleImagePath = v.OperatorBundleImagePath
	col.BundleName = v.OperatorBundleName
	col.DefaultChannel = v.DefaultChannel
	col.Channels = v.Channels
	col.AuditErrors = v.Errors
	col.SkipRange = v.SkipRangeDB
	col.Replace = v.ReplacesDB
	col.BundleVersion = v.VersionDB
	col.OCPLabel = v.OCPLabel
	col.BundleImageBuildDate = v.BuildAt
	col.HasCustomScorecardTests = v.HasCustomScorecardTests
	col.IsHeadOfChannel = v.IsHeadOfChannel

	var csv *v1alpha1.ClusterServiceVersion
	if v.Bundle != nil && v.Bundle.CSV != nil {
		csv = v.Bundle.CSV
	} else if v.CSVFromIndexDB != nil {
		csv = v.CSVFromIndexDB
	}

	col.AddDataFromCSV(csv)
	col.AddDataFromBundle(v.Bundle)
	col.AddDataFromScorecard(v.ScorecardResults)
	col.AddDataFromValidators(v.ValidatorsResults)
	col.SetMaxOpenshiftVersion(csv, v.PropertiesDB)

	if len(col.BundleVersion) < 1 && len(v.VersionDB) > 0 {
		col.BundleVersion = v.VersionDB
	}

	if len(col.BundleVersion) > 0 {
		_, err := semver.Parse(col.BundleVersion)
		if err != nil {
			col.InvalidVersioning = pkg.GetYesOrNo(true)
		} else {
			col.InvalidVersioning = pkg.GetYesOrNo(false)
		}
	}

	if len(col.SkipRange) > 0 {
		_, err := semver.ParseRange(col.SkipRange)
		if err != nil {
			col.InvalidSkipRange = pkg.GetYesOrNo(true)
		} else {
			col.InvalidSkipRange = pkg.GetYesOrNo(false)
		}
	}
	return &col
}

func (c *Column) SetMaxOpenshiftVersion(csv *v1alpha1.ClusterServiceVersion, propertiesDB []pkg.PropertiesAnnotation) {

	if csv == nil {
		return
	}

	cvsProperties := csv.Annotations[olmproperties]
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

	for _, v := range propertiesDB {
		if v.Type == olmmaxOpenShiftVersion {
			c.MaxOCPVersion = v.Value
			break
		}
	}
}

func (c *Column) AddDataFromCSV(csv *v1alpha1.ClusterServiceVersion) {

	if csv == nil {
		return
	}

	certified := csv.ObjectMeta.Annotations[certifiedAnnotation]
	c.Certified = len(certified) > 0 && certified == "true"
	c.Repository = csv.ObjectMeta.Annotations[repositoryAnnotation]
	if len(csv.Spec.Version.String()) > 0 {
		c.BundleVersion = csv.Spec.Version.String()
	}
	c.HasWebhook = len(csv.Spec.WebhookDefinitions) > 0
	c.Maturity = csv.Spec.Maturity
	c.Capabilities = csv.ObjectMeta.Annotations["capabilities"]
	c.Categories = csv.ObjectMeta.Annotations["categories"]

	for k, v := range csv.ObjectMeta.Labels {
		if strings.Contains(k, archLabels) && v == "supported" {
			c.MultipleArchitectures = append(c.MultipleArchitectures, strings.Split(k, archLabels)[1])
		}
		if strings.Contains(k, osLabel) && v == "supported" {
			c.MultipleArchitectures = append(c.MultipleArchitectures, strings.Split(k, osLabel)[1])
		}
	}

	builder := csv.ObjectMeta.Annotations[sdkBuilderAnnotation]
	if len(builder) > 0 {
		c.Builder = builder
		version := strings.Split(builder, "v")
		if len(version) > 1 {
			c.SDKVersion = version[1]
		}
	}

	c.Infrastructure = csv.ObjectMeta.Annotations[infrastructureAnnotation]

	if len(c.Infrastructure) > 0 && len(c.MultipleArchitectures) > 0 {
		c.HasPossiblePerformIssues = true
	}

	if len(csv.ObjectMeta.Annotations[sdkProjectLayoutAnnotation]) > 0 {
		c.ProjectLayout = csv.ObjectMeta.Annotations[sdkProjectLayoutAnnotation]
	}
	skipFromAnnotation := csv.ObjectMeta.Annotations[skipRangeAnnotation]
	if len(skipRangeAnnotation) > 0 {
		c.SkipRange = skipFromAnnotation
	}

	if len(csv.Spec.Replaces) > 0 {
		c.Replace = csv.Spec.Replaces
	}
	c.Skips = csv.Spec.Skips

	for _, v := range csv.Spec.InstallModes {
		if v.Supported {
			switch v.Type {
			case v1alpha1.InstallModeTypeAllNamespaces:
				c.IsSupportingAllNamespaces = true
			case v1alpha1.InstallModeTypeMultiNamespace:
				c.IsSupportingMultiNamespaces = true
			case v1alpha1.InstallModeTypeOwnNamespace:
				c.IsSupportingOwnNamespaces = true
			case v1alpha1.InstallModeTypeSingleNamespace:
				c.IsSupportingSingleNamespace = true
			}
		}
	}

	for _, v := range csv.Spec.Maintainers {
		c.MaintainersEmail = append(c.MaintainersEmail, v.Email)
	}
	c.MaintainersEmail = pkg.GetUniqueValues(c.MaintainersEmail)

	for _, v := range csv.Spec.Links {
		c.Links = append(c.Links, v.URL)
	}
	c.Links = pkg.GetUniqueValues(c.Links)
}

func (c *Column) AddDataFromBundle(bundle *apimanifests.Bundle) {
	if bundle == nil {
		c.KindsDeprecateAPIs = []string{pkg.Unknown}
		return
	}

	removedAPIs := pkg.GetRemovedAPIsOn1_22From(bundle)
	c.KindsDeprecateAPIs = pkg.RemovedAPIsKind(removedAPIs)
	c.DeprecateAPIsManifests = removedAPIs

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

func (c *Column) AddDataFromValidators(validatorsResults []validationerrors.ManifestResult) {
	for _, result := range validatorsResults {
		for _, err := range result.Errors {
			c.ValidatorErrors = append(c.ValidatorErrors, err.Detail)
		}
		for _, err := range result.Warnings {
			c.ValidatorWarnings = append(c.ValidatorWarnings, err.Detail)
		}
	}
}
