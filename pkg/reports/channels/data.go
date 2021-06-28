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
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
)

type Data struct {
	AuditChannel      []models.AuditChannel
	Flags             BindFlags
	IndexImageInspect pkg.DockerInspectManifest
}

func (d *Data) PrepareReport() Report {
	var allColumns []Column
	for _, auditCha := range d.AuditChannel {
		allColumns = append(allColumns, *NewColumn(auditCha))
	}

	sort.Slice(allColumns[:], func(i, j int) bool {
		return allColumns[i].PackageName < allColumns[j].PackageName
	})

	finalReport := Report{}
	finalReport.Flags = d.Flags
	finalReport.Columns = allColumns
	finalReport.IndexImageInspect = d.IndexImageInspect

	dt := time.Now().Format("2006-01-02")
	finalReport.GenerateAt = dt

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
