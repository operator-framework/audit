// Copyright 2023 The Audit Authors
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

package validation

import (
	"fmt"
	"strings"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation/errors"
	interfaces "github.com/operator-framework/api/pkg/validation/interfaces"
)

// WorksWithMicroShiftAPIsValidator will check the bundle for RBAC permissions
// and flag usage of non-compliant Kubernetes API groups.
var WorksWithMicroShiftAPIsValidator interfaces.Validator = interfaces.ValidatorFunc(validateWorksWithMicroShiftAPIs)

func validateWorksWithMicroShiftAPIs(objs ...interface{}) (results []errors.ManifestResult) {
	for _, obj := range objs {
		switch v := obj.(type) {
		case *manifests.Bundle:
			results = append(results, validateAPIGroups(v))
		}
	}

	return results
}

func validateAPIGroups(bundle *manifests.Bundle) errors.ManifestResult {
	result := errors.ManifestResult{}
	if bundle == nil {
		result.Add(errors.ErrInvalidBundle("Bundle is nil", nil))
		return result
	}
	result.Name = bundle.Name

	if bundle.CSV == nil {
		result.Add(errors.ErrInvalidBundle("Bundle csv is nil", bundle.Name))
		return result
	}

	errs := checkAPIGroups(bundle)
	result.Add(errs...)

	return result
}

func checkAPIGroups(bundle *manifests.Bundle) []errors.Error {
	var errs []errors.Error

	allPermissions := append(bundle.CSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions, bundle.CSV.Spec.InstallStrategy.StrategySpec.Permissions...)
	for _, perm := range allPermissions {
		for _, rule := range perm.Rules {
			for _, apiGroup := range rule.APIGroups {
				if !isValidAPIGroup(apiGroup) {
					errs = append(errs, errors.WarnFailedValidation(fmt.Sprintf("Found API group usages not compatible with MicroShift: %s", apiGroup), bundle.Name))
				}
			}
		}
	}

	return errs
}

func isValidAPIGroup(apiGroup string) bool {
	// Allow empty apiGroup, which refers to the core API group in Kubernetes
	if apiGroup == "" {
		return true
	}
	return strings.HasSuffix(apiGroup, ".k8s.io") ||
		apiGroup == "route.openshift.io" ||
		apiGroup == "securitycontextconstraints.openshift.io"
}
