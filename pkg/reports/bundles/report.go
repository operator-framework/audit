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
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/operator-framework/audit/pkg"
)

type Report struct {
	Columns []Columns
	Flags   BindFlags
}

//todo: fix the complexity
//nolint:gocyclo
func (r *Report) writeXls() error {
	const sheetName = "Sheet1"
	f := excelize.NewFile()

	styleOrange, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#ec8f1c",
		},
	})

	styleWrapText, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			WrapText: true,
		},
	})

	columns := map[string]string{
		"A":  "Package Name",
		"B":  "Repository",
		"C":  "Links",
		"D":  "Maturity",
		"E":  "Capabilities",
		"F":  "Categories",
		"G":  "Multiple Architectures",
		"H":  "Certified",
		"I":  "Has v1beta1 CRD?",
		"J":  "Company",
		"K":  "Maintainer Name(s)",
		"L":  "Maintainer Email(s)",
		"M":  "Operator Bundle Name",
		"N":  "Operator Bundle Version",
		"O":  "Default Channel",
		"P":  "Bundle Channel",
		"Q":  "Build At",
		"R":  "Bundle Path",
		"S":  "Has webhooks?",
		"T":  "Builder",
		"U":  "SDK Version",
		"V":  "Project Layout",
		"W":  "Scorecard Failing Tests",
		"X":  "Scorecard Suggestions",
		"Y":  "Scorecard Errors",
		"Z":  "Validator Errors",
		"AA": "Validator Warnings",
		"AB": "Invalid Versioning",
		"AC": "Invalid SkipRange",
		"AD": "Found Replace",
		"AE": "Has Dependency",
		"AF": "Skip Range",
		"AG": "Skips",
		"AH": "Replace",
		"AI": "Supports All Namespaces",
		"AJ": "Supports Single Namespaces",
		"AK": "Supports Own Namespaces",
		"AL": "Supports Multi Namespaces",
		"AM": "Infrastructure",
		"AN": "Has possible performance issues",
		"AO": "OCP Labels Version",
		"AP": "Issues (To process this report)",
	}

	// Header
	dt := time.Now().Format("2006-01-02")
	_ = f.SetCellValue(sheetName, "A1",
		fmt.Sprintf("Audit Bundle Level (%s)", dt))
	_ = f.SetCellValue(sheetName, "A2", "Image used")
	_ = f.SetCellValue(sheetName, "B2", r.Flags.IndexImage)

	for k, v := range columns {
		_ = f.SetCellValue(sheetName, fmt.Sprintf("%s3", k), v)
	}

	for k, v := range r.Columns {
		line := k + 4

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", line), v.PackageName); err != nil {
			log.Errorf("to add packageName cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", line), v.Repository); err != nil {
			log.Errorf("to add repository cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", line),
			pkg.GetFormatArrayWithBreakLine(v.Links)); err != nil {
			log.Errorf("to add links cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("D%d", line), v.Maturity); err != nil {
			log.Errorf("to add Maturity cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("E%d", line), v.Capabilities); err != nil {
			log.Errorf("to add Capabilities cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("F%d", line), v.Categories); err != nil {
			log.Errorf("to add Categories cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("G%d", line),
			pkg.GetFormatArrayWithBreakLine(v.MultipleArchitectures)); err != nil {
			log.Errorf("to add MultipleArchitectures cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("H%d", line), pkg.GetYesOrNo(v.Certified)); err != nil {
			log.Errorf("to add Certified cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("I%d", line), v.HasV1beta1CRDs); err != nil {
			log.Errorf("to add HasV1beta1CRDs cell value : %s", err)
		}
		if v.HasV1beta1CRDs == pkg.GetYesOrNo(true) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("I%d", line),
				fmt.Sprintf("I%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("J%d", line), v.Company); err != nil {
			log.Errorf("to add Company cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("K%d", line),
			pkg.GetFormatArrayWithBreakLine(v.NameMaintainers)); err != nil {
			log.Errorf("to add NameMaintainers cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("L%d", line),
			pkg.GetFormatArrayWithBreakLine(v.EmailMaintainers)); err != nil {
			log.Errorf("to add EmailMaintainers cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("M%d", line), v.OperatorBundleName); err != nil {
			log.Errorf("to add OperatorBundleName cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("N%d", line), v.OperatorBundleVersion); err != nil {
			log.Errorf("to add OperatorBundleVersion cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("O%d", line), v.DefaultChannel); err != nil {
			log.Errorf("to add DefaultChannel cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("P%d", line), v.BundleChannel); err != nil {
			log.Errorf("to add BundleChannel cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("Q%d", line), v.BuildAt); err != nil {
			log.Errorf("to add BuildAt cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("R%d", line), v.BundlePath); err != nil {
			log.Errorf("to add BundlePath cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("S%d", line),
			pkg.GetYesOrNo(v.HasWebhook)); err != nil {
			log.Errorf("to add HasWebhook cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("T%d", line), v.Builder); err != nil {
			log.Errorf("to add Builder cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("U%d", line), v.SDKVersion); err != nil {
			log.Errorf("to add SDKVersion cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("V%d", line), v.ProjectLayout); err != nil {
			log.Errorf("to add ProjectLayout cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("W%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardFailingTests)); err != nil {
			log.Errorf("to add ScorecardFailingTests cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("X%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardSuggestions)); err != nil {
			log.Errorf("to add ScorecardSuggestions cell value : %s", err)
		}
		if len(v.ScorecardSuggestions) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("X%d", line),
				fmt.Sprintf("X%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("Y%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardErrors)); err != nil {
			log.Errorf("to add ScorecardErrors cell value : %s", err)
		}
		if len(v.ScorecardErrors) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Y%d", line),
				fmt.Sprintf("Y%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("Z%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ValidatorErrors)); err != nil {
			log.Errorf("to add ValidatorErrors cell value : %s", err)
		}
		if len(v.ValidatorErrors) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Z%d", line),
				fmt.Sprintf("Z%d", line), styleOrange)
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("W%d", line), fmt.Sprintf("Z%d", line), styleWrapText); err != nil {
			log.Errorf("unable to set style : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AA%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ValidatorWarnings)); err != nil {
			log.Errorf("to add ValidatorWarnings cell value : %s", err)
		}
		// format for the list of issues
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("X%d", line), fmt.Sprintf("AA%d", line), styleWrapText); err != nil {
			log.Errorf("unable to set style : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AB%d", line),
			v.InvalidVersioning); err != nil {
			log.Errorf("to add InvalidVersioning cell value : %s", err)
		}
		if v.InvalidVersioning == pkg.GetYesOrNo(true) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AB%d", line),
				fmt.Sprintf("AB%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AC%d", line),
			v.InvalidSkipRange); err != nil {
			log.Errorf("to add GetYesOrNo cell value : %s", err)
		}
		if v.InvalidSkipRange == pkg.GetYesOrNo(true) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AC%d", line),
				fmt.Sprintf("AC%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AD%d", line),
			v.FoundReplace); err != nil {
			log.Errorf("to add FoundReplace cell value : %s", err)
		}
		if v.FoundReplace == pkg.GetYesOrNo(false) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AD%d", line),
				fmt.Sprintf("AD%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AE%d", line),
			pkg.GetYesOrNo(v.HasDependency)); err != nil {
			log.Errorf("to add HasDependency cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AF%d", line),
			v.SkipRange); err != nil {
			log.Errorf("to add SkipRange cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AG%d", line),
			pkg.GetFormatArrayWithBreakLine(v.Skips)); err != nil {
			log.Errorf("to add HasGKVDependency cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AH%d", line),
			v.Replace); err != nil {
			log.Errorf("to add Replace cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AI%d", line),
			pkg.GetYesOrNo(v.IsSupportingAllNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AJ%d", line),
			pkg.GetYesOrNo(v.IsSupportingSingleNamespace)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AK%d", line),
			pkg.GetYesOrNo(v.IsSupportingOwnNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AL%d", line),
			pkg.GetYesOrNo(v.IsSupportingMultiNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AM%d", line),
			v.Infrastructure); err != nil {
			log.Errorf("to add HasInfraSupport cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AN%d", line),
			pkg.GetYesOrNo(v.HasPossiblePerformIssues)); err != nil {
			log.Errorf("to add HasPossiblePerformIssues cell value : %s", err)
		}
		if v.HasPossiblePerformIssues {
			if err := f.AddComment(sheetName, fmt.Sprintf("AN%d", line),
				fmt.Sprintf(`{"author":"Audit: ","text":"Project using different infracsture (%s) 
				for disconnected scenarios and supporting multi-arch(s) (%s)"}`,
					v.Infrastructure, v.MultipleArchitectures)); err != nil {
				log.Errorf("to add comment for HasPossiblePerformIssues: %s", err)
			}

			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AN%d", line),
				fmt.Sprintf("AK%d", line), styleOrange)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AO%d", line), v.OCPLabel); err != nil {
			log.Errorf("to add OCPLabel cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AP%d", line), v.AuditErrors); err != nil {
			log.Errorf("to add AuditErrors cell value : %s", err)
		}
	}

	// Remove the scorecard columns when that is disable
	if r.Flags.DisableScorecard {
		if err := f.SetColVisible(sheetName, "W", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "X", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "Y", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
	}

	// Found replace when it is not looking all bundles
	if r.Flags.HeadOnly || r.Flags.Limit > 0 {
		if err := f.SetColVisible(sheetName, "AD", false); err != nil {
			log.Errorf("unable to remove found Replace columns : %s", err)
		}
	}

	// Remove the validators columns when that is disable
	if r.Flags.DisableValidators {
		if err := f.SetColVisible(sheetName, "Z", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "AA", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "AB", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
	}

	if err := f.AddTable(sheetName, "A3", "AP3", pkg.TableFormat); err != nil {
		log.Errorf("unable to add table format : %s", err)
	}

	reportFilePath := filepath.Join(r.Flags.OutputPath,
		pkg.GetReportName(r.Flags.IndexImage, "bundles", "xlsx"))

	if err := f.SaveAs(reportFilePath); err != nil {
		return err
	}
	return nil
}

func (r *Report) writeJSON() error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	const reportType = "bundles"
	return pkg.WriteJSON(data, r.Flags.IndexImage, r.Flags.OutputPath, reportType)
}
