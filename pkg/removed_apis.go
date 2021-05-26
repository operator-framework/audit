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

package pkg

import (
	"fmt"
	"strings"

	"github.com/blang/semver"

	"github.com/operator-framework/api/pkg/manifests"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// nolint:gocyclo
// Note that the following code is the same present in the operator-framework/api
// getRemovedAPIsOn1_22From return the list of resources which were deprecated
// and are no longer be supported in 1.22.
// More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22
func GetRemovedAPIsOn1_22From(bundle *manifests.Bundle) map[string][]string {
	deprecatedAPIs := make(map[string][]string)
	if len(bundle.V1beta1CRDs) > 0 {
		var crdAPINames []string
		for _, obj := range bundle.V1beta1CRDs {
			crdAPINames = append(crdAPINames, obj.Name)
		}
		deprecatedAPIs["CRD"] = crdAPINames
	}

	for _, obj := range bundle.Objects {
		switch u := obj.GetObjectKind().(type) {
		case *unstructured.Unstructured:
			switch u.GetAPIVersion() {
			case "scheduling.k8s.io/v1beta1":
				if u.GetKind() == "PriorityClass" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "rbac.authorization.k8s.io/v1beta1":
				if u.GetKind() == "Role" || u.GetKind() == "ClusterRoleBinding" ||
					u.GetKind() == "RoleBinding" || u.GetKind() == "ClusterRole" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "apiregistration.k8s.io/v1beta1":
				if u.GetKind() == "APIService" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "authentication.k8s.io/v1beta1":
				if u.GetKind() == "TokenReview" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "authorization.k8s.io/v1beta1":
				if u.GetKind() == "LocalSubjectAccessReview" || u.GetKind() == "SelfSubjectAccessReview" ||
					u.GetKind() == "SubjectAccessReview" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "admissionregistration.k8s.io/v1beta1":
				if u.GetKind() == "MutatingWebhookConfiguration" ||
					u.GetKind() == "ValidatingWebhookConfiguration" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "coordination.k8s.io/v1beta1":
				if u.GetKind() == "Lease" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "extensions/v1beta1":
				if u.GetKind() == "Ingress" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "networking.k8s.io/v1beta1":
				if u.GetKind() == "Ingress" || u.GetKind() == "IngressClass" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "storage.k8s.io/v1beta1":
				if u.GetKind() == "CSIDriver" || u.GetKind() == "CSINode" ||
					u.GetKind() == "StorageClass" || u.GetKind() == "VolumeAttachment" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			case "certificates.k8s.io/v1beta1":
				if u.GetKind() == "CertificateSigningRequest" {
					deprecatedAPIs[u.GetKind()] = append(deprecatedAPIs[u.GetKind()], obj.GetName())
				}
			}
		}
	}
	return deprecatedAPIs
}

// generateMessageWithDeprecatedAPIs will return a list with the kind and the name
// of the resource which were found and required to be upgraded
func GenerateMessageWithDeprecatedAPIs(deprecatedAPIs map[string][]string) string {
	msg := ""
	count := 0
	for k, v := range deprecatedAPIs {
		if count == len(deprecatedAPIs)-1 {
			msg = msg + fmt.Sprintf("%s: (%+q)", k, v)
		} else {
			msg = msg + fmt.Sprintf("%s: (%+q),", k, v)
		}
	}
	return msg
}

// RemovedAPIsKind return a list with all Kinds found
func RemovedAPIsKind(deprecatedAPIs map[string][]string) []string {
	var list []string
	for k := range deprecatedAPIs {
		list = append(list, k)
	}
	return list
}

// OCP version where the apis v1beta1 is no longer supported
const ocpVerV1beta1Unsupported = "4.9"

// IsComplyingWithDeprecatedCriteria will verify if the OpenShiftVersion property was informed as the OCP label index
// For audit we have not the dockerfile so we are checking by here.
func IsComplyingWithDeprecatedCriteria(maxOCPVersion, ocpLabel string) bool {
	return IsMaxOCPVersionLowerThan49(maxOCPVersion) && IsOcpLabelRangeLowerThan49(ocpLabel)
}

func IsMaxOCPVersionLowerThan49(maxOCPVersion string) bool {
	if len(maxOCPVersion) == 0 {
		return false
	}
	semVerVersionMaxOcp, err := semver.ParseTolerant(maxOCPVersion)
	if err != nil {
		return false
	}

	semVerOCPV1beta1Unsupported, _ := semver.ParseTolerant(ocpVerV1beta1Unsupported)
	if semVerVersionMaxOcp.GE(semVerOCPV1beta1Unsupported) {
		return false
	}
	return false
}

// IsOcpLabelRangeLowerThan49 returns true if the range < 4.9
func IsOcpLabelRangeLowerThan49(ocpLabel string) bool {
	if len(ocpLabel) == 0 {
		return false
	}
	semVerOCPV1beta1Unsupported, _ := semver.ParseTolerant(ocpVerV1beta1Unsupported)
	if strings.Contains(ocpLabel, "=") {
		version := strings.Split(ocpLabel, "=")[1]
		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			return false
		}
		if verParsed.GE(semVerOCPV1beta1Unsupported) {
			return false
		}
	} else {
		// if not has not the = then the value needs contains - value less < 4.9
		if !strings.Contains(ocpLabel, "-") {
			return false
		}
		version := strings.Split(ocpLabel, "-")[1]
		verParsed, err := semver.ParseTolerant(version)
		if err != nil {
			return false
		}
		if verParsed.GE(semVerOCPV1beta1Unsupported) {
			return false
		}
	}
	return true
}
