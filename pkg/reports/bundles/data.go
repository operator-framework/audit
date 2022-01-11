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

package bundles

import (
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	sq "github.com/Masterminds/squirrel"
	"github.com/operator-framework/audit/pkg"

	"github.com/operator-framework/audit/pkg/models"
)

type Data struct {
	AuditBundle       []models.AuditBundle
	Flags             BindFlags
	IndexImageInspect pkg.DockerInspect
}

func (d *Data) PrepareReport() Report {
	d.fixPackageNameInconsistency()

	var allColumns []Column
	for _, v := range d.AuditBundle {
		col := NewColumn(v)

		// do not add bundle which has not the label
		if len(d.Flags.Label) > 0 && !v.FoundLabel {
			continue
		}

		allColumns = append(allColumns, *col)
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

// fix inconsistency in the index db
// some packages are empty then, we get them by looking for the bundles
// which are publish with the same registry path
func (d *Data) fixPackageNameInconsistency() {
	for _, auditBundle := range d.AuditBundle {
		if auditBundle.PackageName == "" {
			split := strings.Split(auditBundle.OperatorBundleImagePath, "/")
			nm := ""
			for _, v := range split {
				if strings.Contains(v, "@") {
					nm = strings.Split(v, "@")[0]
					break
				}
			}
			for _, bundle := range d.AuditBundle {
				if strings.Contains(bundle.OperatorBundleImagePath, nm) {
					auditBundle.PackageName = bundle.PackageName
				}
			}
		}
	}
}

func (d *Data) OutputReport() error {
	report := d.PrepareReport()
	switch d.Flags.OutputFormat {
	case pkg.JSON:
		if err := report.writeJSON(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid output format : %s", d.Flags.OutputFormat)
	}
	return nil
}

func (d *Data) BuildBundlesQuery() (string, error) {
	query := sq.Select("o.name, o.csv, o.bundlepath").From(
		"operatorbundle o")

	if d.Flags.HeadOnly {
		query = sq.Select("o.name, o.csv, o.bundlepath").From(
			"operatorbundle o, channel c")
		query = query.Where("c.head_operatorbundle_name == o.name")
	}
	if d.Flags.Limit > 0 {
		query = query.Limit(uint64(d.Flags.Limit))
	}
	if len(d.Flags.Filter) > 0 {
		query = sq.Select("o.name, o.csv, o.bundlepath").From(
			"operatorbundle o, channel_entry c")
		like := "'%" + d.Flags.Filter + "%'"
		query = query.Where(fmt.Sprintf("c.operatorbundle_name == o.name AND c.package_name like %s", like))
	}

	query.OrderBy("o.name")

	sql, _, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("unable to create sql : %s", err)
	}
	return sql, nil
}
