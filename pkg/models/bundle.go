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

package models

import (
	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/validation/errors"
	"github.com/operator-framework/audit/pkg"
)

// AuditBundle defines the data per bundle which is gathering to generate the reports
type AuditBundle struct {
	Bundle                  *apimanifests.Bundle
	FoundLabel              bool
	OperatorBundleName      string
	OperatorBundleImagePath string
	PackageName             string
	DefaultChannel          string
	ScorecardResults        v1alpha3.TestList
	ValidatorsResults       []errors.ManifestResult
	CSVFromIndexDB          *v1alpha1.ClusterServiceVersion
	PropertiesDB            []pkg.PropertiesAnnotation
	Channels                []string
	HasCustomScorecardTests bool
	IsHeadOfChannel         bool
	BundleImageLabels       map[string]string `json:"bundleImageLabels,omitempty"`
	BundleAnnotations       map[string]string `json:"bundleAnnotations,omitempty"`
	Errors                  []string
}

func NewAuditBundle(operatorBundleName, operatorBundleImagePath string) *AuditBundle {
	auditBundle := AuditBundle{}
	auditBundle.OperatorBundleName = operatorBundleName
	auditBundle.OperatorBundleImagePath = operatorBundleImagePath

	return &auditBundle
}
