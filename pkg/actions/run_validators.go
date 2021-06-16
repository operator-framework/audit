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
	apivalidation "github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/api/pkg/validation/errors"
	"github.com/operator-framework/audit/pkg/models"
)

func RunValidators(auditBundle *models.AuditBundle) *models.AuditBundle {
	validators := apivalidation.DefaultBundleValidators
	validators = validators.WithValidators(apivalidation.OperatorHubValidator)
	validators = validators.WithValidators(apivalidation.ObjectValidator)
	// todo: check how can we call the community validator since it will make all bundles
	// shipped previously fail
	// validators = validators.WithValidators(apivalidation.CommunityOperatorValidator)

	objs := auditBundle.Bundle.ObjectsToValidate()

	results := validators.Validate(objs...)
	nonEmptyResults := []errors.ManifestResult{}

	for _, result := range results {
		if result.HasError() || result.HasWarn() {
			nonEmptyResults = append(nonEmptyResults, result)
		}
	}

	auditBundle.ValidatorsResults = nonEmptyResults
	return auditBundle
}
