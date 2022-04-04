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

package validation

import (
	"fmt"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation/errors"
	interfaces "github.com/operator-framework/api/pkg/validation/interfaces"
)

// BundleSizeValidator will check the bundle size according to its limits
// note that this check will raise an error if the size is bigger than the max allowed
// and warnings when:
// - we are unable to check the bundle size because we are running a check without load the bundle
// - we could identify that the bundle size is close to the limit (bigger than 85%)
// - [Deprecated and planned to be removed at 2023 -  The API will start growing to encompass validation for all past
// history] if the bundle size uncompressed < ~1MB and it cannot work on clusters which uses OLM versions < 1.17.5
// todo: remove this check when OCP 4.8 be in EOL.
var BundleSizeValidator interfaces.Validator = interfaces.ValidatorFunc(validateBundleSizeValidator)

// maxBundleSize is the maximum size of a bundle in bytes.
// This ensures the bundle can be staged in a single ConfigMap by OLM during installation.
// The value is derived from the standard upper bound for k8s resources (~1MB).
// We will use this value to check the bundle compressed is < ~1MB
const maxBundleSize = int64(1 << (10 * 2))

func validateBundleSizeValidator(objs ...interface{}) (results []errors.ManifestResult) {

	for _, obj := range objs {
		switch v := obj.(type) {
		case *manifests.Bundle:
			results = append(results, validateBundleSize(v))
		}
	}

	return results
}

// validateBundleSize will check the bundle size is bigger than > 1 MB ( valid up to OCP 4.9 )
// After 4.9 we begin to compress then it is no longer an issue.
func validateBundleSize(bundle *manifests.Bundle) errors.ManifestResult {
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

	errors := checkBundleSize(bundle)
	result.Add(errors...)

	return result
}

func checkBundleSize(bundle *manifests.Bundle) []errors.Error {
	var errs []errors.Error

	if bundle.Size == 0 {
		errs = append(errs, errors.WarnFailedValidation("unable to check the bundle size", bundle.Name))
		return errs
	}

	// @Deprecated
	// Before these versions the bundles were not compressed
	// and their size must be < ~1MB
	if bundle.Size > maxBundleSize {
		errs = append(errs, errors.ErrInvalidBundle(
			fmt.Sprintf("bundle uncompressed size exceeded the limit support for OLM versions relesed prior"+
				" 1.17.5 :  size=~%s , max=%s. "+
				"(these bundle cannot work in any cluster or vendor which uses OLM versions < 1.17.5 and OpenShift "+
				"versions < 4.9)",
				formatBytesInUnit(bundle.Size),
				formatBytesInUnit(maxBundleSize)),
			bundle.Name))
	}

	return errs
}

func formatBytesInUnit(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
