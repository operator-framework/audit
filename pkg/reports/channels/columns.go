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

package channels

import (
	"strings"

	"github.com/blang/semver"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Column struct {
	PackageName               string   `json:"packageName"`
	ChannelName               string   `json:"channelName"`
	IsUsingSkips              bool     `json:"isUsingSkips,omitempty"`
	IsUsingSkipRange          bool     `json:"isUsingSkipRange,omitempty"`
	IsFollowingNameConvention bool     `json:"isFollowingNameConvention,omitempty"`
	HasInvalidSkipRange       bool     `json:"HasInvalidSkipRange,omitempty"`
	HasInvalidVersioning      bool     `json:"HasInvalidVersioning,omitempty"`
	AuditErrors               []string `json:"errors,omitempty"`
}

func NewColumn(auditCha models.AuditChannel) *Column {
	col := Column{}
	col.PackageName = auditCha.PackageName
	col.ChannelName = auditCha.ChannelName
	col.IsFollowingNameConvention = isFollowingConventional(auditCha.ChannelName)

	var allBundles []bundles.Column
	for _, v := range auditCha.AuditBundles {
		bundles := bundles.Column{}
		bundles.Replace = v.ReplacesDB
		bundles.SkipRange = v.SkipRangeDB
		bundles.PackageName = v.PackageName
		bundles.Channels = v.Channels
		if len(v.SkipsDB) > 0 {
			bundles.Skips = strings.Split(v.SkipsDB, ",")
		}

		if len(v.VersionDB) > 0 {
			_, err := semver.Parse(v.VersionDB)
			if err != nil {
				bundles.InvalidVersioning = pkg.GetYesOrNo(true)
			} else {
				bundles.InvalidVersioning = pkg.GetYesOrNo(false)
			}
		}

		if len(v.SkipRangeDB) > 0 {
			_, err := semver.ParseRange(v.SkipRangeDB)
			if err != nil {
				bundles.InvalidSkipRange = pkg.GetYesOrNo(true)
			} else {
				bundles.InvalidSkipRange = pkg.GetYesOrNo(false)
			}
		}
	}

	var auditErrors []string

	foundInvalidSkipRange := false
	foundInvalidVersioning := false

	for _, v := range allBundles {
		if !foundInvalidVersioning && v.InvalidVersioning == pkg.GetYesOrNo(true) {
			foundInvalidVersioning = true
		}
		if !foundInvalidSkipRange && len(v.InvalidSkipRange) > 0 && v.InvalidSkipRange == pkg.GetYesOrNo(true) {
			foundInvalidSkipRange = true
		}
	}

	col.HasInvalidVersioning = foundInvalidVersioning
	col.HasInvalidSkipRange = foundInvalidSkipRange
	col.AuditErrors = auditErrors
	return &col

}
