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
	"strings"

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

type Columns struct {
	PackageName                 string   `json:"packageName"`
	OperatorBundleName          string   `json:"operatorBundleName"`
	OperatorBundleVersion       string   `json:"operatorBundleVersion,omitempty"`
	Certified                   bool     `json:"certified"`
	BundlePath                  string   `json:"bundlePath,omitempty"`
	HasWebhook                  bool     `json:"hasWebhook"`
	HasV1beta1CRDs              string   `json:"hasV1beta1CRDs,omitempty"`
	BuildAt                     string   `json:"buildAt,omitempty"`
	Company                     string   `json:"company,omitempty"`
	Repository                  string   `json:"repository,omitempty"`
	BundleChannel               string   `json:"bundleChannel,omitempty"`
	DefaultChannel              string   `json:"defaultChannel,omitempty"`
	Maturity                    string   `json:"maturity,omitempty"`
	EmailMaintainers            []string `json:"emailMaintainers,omitempty"`
	NameMaintainers             []string `json:"nameMaintainers,omitempty"`
	Links                       []string `json:"links,omitempty"`
	Capabilities                string   `json:"capabilities,omitempty"`
	Categories                  string   `json:"categories,omitempty"`
	MultipleArchitectures       []string `json:"multipleArchitectures,omitempty"`
	Builder                     string   `json:"builder,omitempty"`
	SDKVersion                  string   `json:"sdkVersion,omitempty"`
	ProjectLayout               string   `json:"projectLayout,omitempty"`
	ValidatorErrors             []string `json:"validatorErrors,omitempty"`
	ValidatorWarnings           []string `json:"validatorWarnings,omitempty"`
	ScorecardErrors             []string `json:"scorecardErrors,omitempty"`
	ScorecardSuggestions        []string `json:"scorecardSuggestions,omitempty"`
	ScorecardFailingTests       []string `json:"scorecardFailingTests,omitempty"`
	InvalidVersioning           string   `json:"invalidVersioning,omitempty"`
	InvalidSkipRange            string   `json:"invalidSkipRange,omitempty"`
	FoundReplace                string   `json:"foundReplace,omitempty"`
	HasDependency               bool     `json:"HasDependency,omitempty"`
	SkipRange                   string   `json:"skipRange,omitempty"`
	Skips                       []string `json:"skips,omitempty"`
	Replace                     string   `json:"replace,omitempty"`
	IsSupportingAllNamespaces   bool     `json:"supportsAllNamespaces,omitempty"`
	IsSupportingMultiNamespaces bool     `json:"supportsMultiNamespaces,omitempty"`
	IsSupportingSingleNamespace bool     `json:"supportSingleNamespaces,omitempty"`
	IsSupportingOwnNamespaces   bool     `json:"supportsOwnNamespaces,omitempty"`
	Infrastructure              string   `json:"infrastructure,omitempty"`
	HasPossiblePerformIssues    bool     `json:"hasPossiblePerformIssues,omitempty"`
	OCPLabel                    string   `json:"ocpLabel,omitempty"`
	AuditErrors                 []error  `json:"auditErrors,omitempty"`
}

func (c *Columns) AddDataFromCSV(csv *v1alpha1.ClusterServiceVersion) {

	if csv == nil {
		return
	}

	certified := csv.ObjectMeta.Annotations[certifiedAnnotation]
	c.Certified = len(certified) > 0 && certified == "true"
	c.Repository = csv.ObjectMeta.Annotations[repositoryAnnotation]
	if len(csv.Spec.Version.String()) > 0 {
		c.OperatorBundleVersion = csv.Spec.Version.String()
	}
	c.Company = csv.Spec.Provider.Name
	c.HasWebhook = len(csv.Spec.WebhookDefinitions) > 0
	c.Maturity = csv.Spec.Maturity
	c.Capabilities = csv.ObjectMeta.Annotations["capabilities"]
	c.Categories = csv.ObjectMeta.Annotations["categories"]
	for _, v := range csv.Spec.Links {
		c.Links = append(c.Links, v.URL)
	}
	for _, v := range csv.Spec.Maintainers {
		c.NameMaintainers = append(c.NameMaintainers, v.Name)
	}
	for _, v := range csv.Spec.Maintainers {
		c.EmailMaintainers = append(c.EmailMaintainers, v.Email)
	}

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
}

func (c *Columns) AddDataFromBundle(bundle *apimanifests.Bundle) {
	if bundle == nil {
		c.HasV1beta1CRDs = pkg.Unknown
		return
	}
	c.HasV1beta1CRDs = pkg.GetYesOrNo(len(bundle.V1beta1CRDs) > 0)
	c.HasDependency = bundle.Dependencies != nil && len(bundle.Dependencies) > 0
}

func (c *Columns) AddDataFromScorecard(scorecardResults v1alpha3.TestList) {
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

func (c *Columns) AddDataFromValidators(validatorsResults []validationerrors.ManifestResult) {
	for _, result := range validatorsResults {
		for _, err := range result.Errors {
			c.ValidatorErrors = append(c.ValidatorErrors, err.Detail)
		}
		for _, err := range result.Warnings {
			c.ValidatorWarnings = append(c.ValidatorWarnings, err.Detail)
		}
	}
}
