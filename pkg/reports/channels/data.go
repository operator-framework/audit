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
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Data struct {
	AuditChannel []models.AuditChannel
	Flags        BindFlags
}

func (d *Data) PrepareReport() Report {
	var allColumns []Columns
	for _, auditCha := range d.AuditChannel {

		col := Columns{}
		col.PackageName = auditCha.PackageName
		col.ChannelName = auditCha.ChannelName
		col.IsFollowingNameConvention = isFollowingConventional(auditCha.ChannelName)

		var allBundles []bundles.Columns
		for _, v := range auditCha.AuditBundles {

			bundles := bundles.Columns{}

			var csv *v1alpha1.ClusterServiceVersion
			if v.Bundle != nil && v.Bundle.CSV != nil {
				csv = v.Bundle.CSV
			} else if v.CSVFromIndexDB != nil {
				csv = v.CSVFromIndexDB
			}

			bundles.AddDataFromCSV(csv)
			bundles.AddDataFromBundle(v.Bundle)
			allBundles = append(allBundles, bundles)
		}

		var auditErrors []error

		foundInvalidSkipRange := false
		foundInvalidVersioning := false

		var missingReplace []string
		for _, v := range allBundles {
			if !foundInvalidVersioning && v.InvalidVersioning == pkg.GetYesOrNo(true) {
				foundInvalidVersioning = true
			}
			if !foundInvalidSkipRange && len(v.InvalidSkipRange) > 0 && v.InvalidSkipRange == pkg.GetYesOrNo(true) {
				foundInvalidSkipRange = true
			}
			if len(v.Replace) > 0 && v.FoundReplace == pkg.GetYesOrNo(false) {
				missingReplace = append(missingReplace, v.Replace)
			}

		}

		col.MissingReplaces = missingReplace
		col.FoundAllReplaces = len(missingReplace) == 0
		col.HasInvalidVersioning = foundInvalidVersioning
		col.HasInvalidSkipRange = foundInvalidSkipRange
		col.AuditErrors = auditErrors
		allColumns = append(allColumns, col)
	}

	finalReport := Report{}
	finalReport.Flags = d.Flags
	finalReport.Columns = allColumns
	return finalReport
}

func (d *Data) OutputReport() error {
	report := d.PrepareReport()
	switch d.Flags.OutputFormat {
	case pkg.Xls:
		if err := report.writeXls(); err != nil {
			return err
		}
	case pkg.JSON:
		if err := report.writeJSON(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format : %s", d.Flags.OutputFormat)
	}
	return nil
}

func (d *Data) BuildChannelsQuery() (string, error) {
	query := sq.Select("name, package_name, " +
		"head_operatorbundle_name").From("channel")

	if d.Flags.Limit > 0 {
		query = query.Limit(uint64(d.Flags.Limit))
	}

	if len(d.Flags.Filter) > 0 {
		like := "'%" + d.Flags.Filter + "%'"
		query = query.Where(fmt.Sprintf("package_name like %s", like))
	}

	query.OrderBy("name")
	sql, _, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("unable to create sql : %s", err)
	}
	return sql, nil
}

// isFollowingConventional will check the channels.
func isFollowingConventional(channel string) bool {
	const preview = "preview"
	const stable = "stable"
	const fast = "fast"

	if !strings.HasPrefix(channel, preview) &&
		!strings.HasPrefix(channel, stable) &&
		!strings.HasPrefix(channel, fast) {
		return false
	}

	return true
}
