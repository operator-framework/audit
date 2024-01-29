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

package actions

import (
	"path/filepath"
	"strings"

	"github.com/operator-framework/audit/pkg/validation"

	apivalidation "github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/api/pkg/validation/errors"
	"github.com/operator-framework/audit/pkg/models"
	ocp "github.com/redhat-openshift-ecosystem/ocp-olm-catalog-validator/pkg/validation"
)

func RunValidators(bundlePath string, auditBundle *models.AuditBundle, indexImage string) *models.AuditBundle {
	checkBundleAgainstCommonCriteria(auditBundle)
	fromOCPValidator(auditBundle, bundlePath)

	// Are there obvious "won't work with MicroShift" APIs in use?
	fromWorksWithMicroShiftAPIsValidator(auditBundle)

	// If the index is < 4.9 then do thw following check
	if strings.Contains(indexImage, "4.6") ||
		strings.Contains(indexImage, "4.7") ||
		strings.Contains(indexImage, "4.8") {
		fromAuditValidatorsBundleSize(auditBundle)
	}

	return auditBundle
}

// checkBundleAgainstCommonCriteria will check the bundle against the criteria defined in the
// https://github.com/operator-framework/api
func checkBundleAgainstCommonCriteria(auditBundle *models.AuditBundle) {
	validators := apivalidation.DefaultBundleValidators
	validators = validators.WithValidators(apivalidation.OperatorHubValidator)
	validators = validators.WithValidators(apivalidation.ObjectValidator)
	validators = validators.WithValidators(apivalidation.AlphaDeprecatedAPIsValidator)
	validators = validators.WithValidators(apivalidation.GoodPracticesValidator)

	objs := auditBundle.Bundle.ObjectsToValidate()
	results := validators.Validate(objs...)
	nonEmptyResults := []errors.ManifestResult{}

	for _, result := range results {
		if result.HasError() || result.HasWarn() {
			nonEmptyResults = append(nonEmptyResults, result)
		}
	}

	auditBundle.ValidatorsResults = append(auditBundle.ValidatorsResults, nonEmptyResults...)
}

// checkBundleAgainstCommonCriteria will check the bundle against the criteria defined in the
// https://github.com/redhat-openshift-ecosystem/ocp-olm-catalog-validator which is OCP
// specific
func fromOCPValidator(auditBundle *models.AuditBundle, bundlePath string) {
	validators := ocp.OpenShiftValidator
	objs := auditBundle.Bundle.ObjectsToValidate()

	nonEmptyResults := []errors.ManifestResult{}

	annotationsPath := filepath.Join(bundlePath, "/metadata/annotations.yaml")

	// Pass the --optional-values. e.g. --optional-values="k8s-version=1.22"
	// or --optional-values="image-path=bundle.Dockerfile"
	//nolint: typecheck
	var optionalValues = map[string]string{
		"file": annotationsPath,
	}
	objs = append(objs, optionalValues)
	results := validators.Validate(objs...)

	for _, result := range results {
		if result.HasError() || result.HasWarn() {
			nonEmptyResults = append(nonEmptyResults, result)
		}
	}

	auditBundle.ValidatorsResults = append(auditBundle.ValidatorsResults, nonEmptyResults...)
}

func fromAuditValidatorsBundleSize(auditBundle *models.AuditBundle) {
	validators := validation.BundleSizeValidator
	objs := auditBundle.Bundle.ObjectsToValidate()

	nonEmptyResults := []errors.ManifestResult{}
	results := validators.Validate(objs...)

	for _, result := range results {
		if result.HasError() || result.HasWarn() {
			nonEmptyResults = append(nonEmptyResults, result)
		}
	}

	auditBundle.ValidatorsResults = append(auditBundle.ValidatorsResults, nonEmptyResults...)
}

func fromWorksWithMicroShiftAPIsValidator(auditBundle *models.AuditBundle) {
	validators := validation.WorksWithMicroShiftAPIsValidator
	objs := auditBundle.Bundle.ObjectsToValidate()

	nonEmptyResults := []errors.ManifestResult{}
	results := validators.Validate(objs...)

	for _, result := range results {
		if result.HasError() || result.HasWarn() {
			nonEmptyResults = append(nonEmptyResults, result)
		}
	}

	auditBundle.ValidatorsResults = append(auditBundle.ValidatorsResults, nonEmptyResults...)
}
