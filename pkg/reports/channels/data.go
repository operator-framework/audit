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
	"log"
	"sort"
	"strings"

	"github.com/blang/semver"

	sq "github.com/Masterminds/squirrel"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type Data struct {
	AuditChannel      []models.AuditChannel
	Flags             BindFlags
	IndexImageInspect pkg.DockerInspectManifest
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

			if len(v.ReplacesDB) > 0 {
				// check if found replace
				bundles.FoundReplace = pkg.GetYesOrNo(false)
				for _, b := range auditCha.AuditBundles {
					if b.OperatorBundleName == bundles.Replace {
						bundles.FoundReplace = pkg.GetYesOrNo(true)
						break
					}
				}
			}
		}

		var auditErrors []string

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

	sort.Slice(allColumns[:], func(i, j int) bool {
		return allColumns[i].PackageName < allColumns[j].PackageName
	})

	finalReport := Report{}
	finalReport.Flags = d.Flags
	finalReport.Columns = allColumns
	finalReport.IndexImageInspect = d.IndexImageInspect

	if len(allColumns) == 0 {
		log.Fatal("No data was found for the criteria informed. " +
			"Please, ensure that you provide valid information.")
	}

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
	case pkg.All:
		if err := report.writeXls(); err != nil {
			return err
		}
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
