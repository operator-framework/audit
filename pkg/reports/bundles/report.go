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
	Columns           []Column
	Flags             BindFlags
	IndexImageInspect pkg.DockerInspectManifest
	GenerateAt        string
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

	styleRed, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#EC1C1C",
		},
	})

	styleGreen, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Color: "#3FA91E",
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
		"C":  "OCP Labels Version",
		"D":  "Maturity",
		"E":  "Capabilities",
		"F":  "Categories",
		"G":  "Multiple Architectures",
		"H":  "Certified",
		"I":  "Kinds (Deprecated APIs on 1.22)",
		"J":  "Operator Bundle Name",
		"K":  "Operator Bundle Version",
		"L":  "Default Channel",
		"M":  "Bundle Channel",
		"N":  "Build Date (from index image)",
		"O":  "Bundle Path",
		"P":  "Has webhooks",
		"Q":  "Builder",
		"R":  "SDK Version",
		"S":  "Project Layout",
		"T":  "Scorecard Failing Tests",
		"U":  "Scorecard Suggestions",
		"V":  "Scorecard Errors",
		"W":  "Validator Errors",
		"X":  "Validator Warnings",
		"Y":  "Invalid Versioning",
		"Z":  "Invalid SkipRange",
		"AA": "Is head of channel",
		"AB": "Skip Range",
		"AC": "Skips",
		"AD": "Replace",
		"AE": "Supports All Namespaces",
		"AF": "Supports Single Namespaces",
		"AG": "Supports Own Namespaces",
		"AH": "Supports Multi Namespaces",
		"AI": "Infrastructure Annotations",
		"AJ": "Has possible performance issues",
		"AK": "Suggestion API(s) manifests",
		"AL": "Max OCP Version",
		"AM": "Has custom Scorecards",
		"AN": "Is Default Channel",
		"AO": "Is Deprecated",
		"AP": "Issues (To process this report)",
	}

	// Header
	dt := time.Now().Format("2006-01-02")
	_ = f.SetCellValue(sheetName, "A1",
		fmt.Sprintf("Audit Bundle Report (Generated at %s)", dt))
	_ = f.SetCellValue(sheetName, "A2", "Image used")
	_ = f.SetCellValue(sheetName, "B2", r.Flags.IndexImage)
	_ = f.SetCellValue(sheetName, "A3", "Image Index Create Date:")
	_ = f.SetCellValue(sheetName, "B3", r.IndexImageInspect.Created)
	_ = f.SetCellValue(sheetName, "A4", "Image Index ID:")
	_ = f.SetCellValue(sheetName, "B4", r.IndexImageInspect.ID)

	for k, v := range columns {
		_ = f.SetCellValue(sheetName, fmt.Sprintf("%s5", k), v)
	}

	for k, v := range r.Columns {
		line := k + 6

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", line), v.PackageName); err != nil {
			log.Errorf("to add packageName cell value: %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", line), v.Repository); err != nil {
			log.Errorf("to add repository cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("C%d", line),
			v.OCPLabel); err != nil {
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
		if err := f.SetCellValue(sheetName, fmt.Sprintf("I%d", line), v.KindsDeprecateAPIs); err != nil {
			log.Errorf("to add v cell value : %s", err)
		}
		if len(v.KindsDeprecateAPIs) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("I%d", line),
				fmt.Sprintf("I%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("J%d", line), v.BundleName); err != nil {
			log.Errorf("to add BundleName cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("K%d", line), v.BundleVersion); err != nil {
			log.Errorf("to add BundleVersion cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("L%d", line), v.DefaultChannel); err != nil {
			log.Errorf("to add DefaultChannel cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("M%d", line),
			pkg.GetFormatArrayWithBreakLine(v.Channels)); err != nil {
			log.Errorf("to add Channels cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("N%d", line), v.BundleImageBuildDate); err != nil {
			log.Errorf("to add BundleImageBuildDate cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("O%d", line), v.BundleImagePath); err != nil {
			log.Errorf("to add BundleImagePath cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("P%d", line),
			pkg.GetYesOrNo(v.HasWebhook)); err != nil {
			log.Errorf("to add HasWebhook cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("Q%d", line), v.Builder); err != nil {
			log.Errorf("to add Builder cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("R%d", line), v.SDKVersion); err != nil {
			log.Errorf("to add SDKVersion cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("S%d", line), v.ProjectLayout); err != nil {
			log.Errorf("to add ProjectLayout cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("T%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardFailingTests)); err != nil {
			log.Errorf("to add ScorecardFailingTests cell value : %s", err)
		}
		if len(v.ScorecardFailingTests) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("T%d", line),
				fmt.Sprintf("T%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("U%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardSuggestions)); err != nil {
			log.Errorf("to add ScorecardSuggestions cell value : %s", err)
		}
		if len(v.ScorecardSuggestions) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("U%d", line),
				fmt.Sprintf("U%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("V%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ScorecardErrors)); err != nil {
			log.Errorf("to add ScorecardErrors cell value : %s", err)
		}
		if len(v.ScorecardErrors) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("V%d", line),
				fmt.Sprintf("V%d", line), styleRed)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("W%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ValidatorErrors)); err != nil {
			log.Errorf("to add ValidatorErrors cell value : %s", err)
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("W%d", line), fmt.Sprintf("W%d", line), styleWrapText); err != nil {
			log.Errorf("unable to set style : %s", err)
		}
		if len(v.ValidatorErrors) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("W%d", line),
				fmt.Sprintf("W%d", line), styleRed)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("X%d", line),
			pkg.GetFormatArrayWithBreakLine(v.ValidatorWarnings)); err != nil {
			log.Errorf("to add ValidatorWarnings cell value : %s", err)
		}
		// format for the list of issues
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("X%d", line), fmt.Sprintf("X%d", line), styleWrapText); err != nil {
			log.Errorf("unable to set style : %s", err)
		}
		if len(v.ValidatorWarnings) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("X%d", line),
				fmt.Sprintf("X%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("Y%d", line),
			v.InvalidVersioning); err != nil {
			log.Errorf("to add InvalidVersioning cell value : %s", err)
		}
		if v.InvalidVersioning == pkg.GetYesOrNo(true) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Y%d", line),
				fmt.Sprintf("Y%d", line), styleOrange)
		} else {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Y%d", line),
				fmt.Sprintf("Y%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("Z%d", line),
			v.InvalidSkipRange); err != nil {
			log.Errorf("to add GetYesOrNo cell value : %s", err)
		}
		if v.InvalidSkipRange == pkg.GetYesOrNo(true) {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("Z%d", line),
				fmt.Sprintf("AC%d", line), styleRed)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AA%d", line),
			pkg.GetYesOrNo(v.IsHeadOfChannel)); err != nil {
			log.Errorf("to add IsHeadOfChannel cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AB%d", line),
			v.SkipRange); err != nil {
			log.Errorf("to add SkipRange cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AC%d", line),
			pkg.GetFormatArrayWithBreakLine(v.Skips)); err != nil {
			log.Errorf("to add HasGKVDependency cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AD%d", line),
			v.Replace); err != nil {
			log.Errorf("to add Replace cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AE%d", line),
			pkg.GetYesOrNo(v.IsSupportingAllNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AF%d", line),
			pkg.GetYesOrNo(v.IsSupportingSingleNamespace)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AG%d", line),
			pkg.GetYesOrNo(v.IsSupportingOwnNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AH%d", line),
			pkg.GetYesOrNo(v.IsSupportingMultiNamespaces)); err != nil {
			log.Errorf("to add HasSupportForAllNamespaces cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AI%d", line),
			v.Infrastructure); err != nil {
			log.Errorf("to add HasInfraAnnotation cell value : %s", err)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AJ%d", line),
			pkg.GetYesOrNo(v.HasPossiblePerformIssues)); err != nil {
			log.Errorf("to add HasPossiblePerformIssues cell value : %s", err)
		}
		if v.HasPossiblePerformIssues {
			if err := f.AddComment(sheetName, fmt.Sprintf("AJ%d", line),
				`{"author":"Audit: ","text":"Project supports Disconnected Mode and Multiple Architectures"}`); err != nil {
				log.Errorf("to add comment for HasPossiblePerformIssues: %s", err)
			}

			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AJ%d", line),
				fmt.Sprintf("AJ%d", line), styleOrange)
		}
		if err := f.SetCellValue(sheetName, fmt.Sprintf("AK%d", line),
			pkg.GenerateMessageWithDeprecatedAPIs(v.DeprecateAPIsManifests)); err != nil {
			log.Errorf("to add DeprecateAPIsManifests cell value : %s", err)
		}
		if len(v.DeprecateAPIsManifests) > 0 {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AK%d", line),
				fmt.Sprintf("AL%d", line), styleOrange)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AL%d", line), v.MaxOCPVersion); err != nil {
			log.Errorf("to add MaxOCPVersion cell value : %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AM%d", line),
			pkg.GetYesOrNo(v.HasCustomScorecardTests)); err != nil {
			log.Errorf("to add HasCustomScorecardTests cell value: %s", err)
		}

		if v.HasCustomScorecardTests {
			_ = f.SetCellStyle(sheetName, fmt.Sprintf("AM%d", line),
				fmt.Sprintf("AM%d", line), styleGreen)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AN%d", line),
			pkg.GetYesOrNo(v.IsFromDefaultChannel)); err != nil {
			log.Errorf("to add IsFromDefaultChannel cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AO%d", line),
			pkg.GetYesOrNo(v.IsDeprecated)); err != nil {
			log.Errorf("to add IsDeprecated cell value: %s", err)
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("AP%d", line), v.AuditErrors); err != nil {
			log.Errorf("to add AuditErrors cell value : %s", err)
		}
	}

	// Remove the scorecard columns when that is disable
	if r.Flags.DisableScorecard {
		if err := f.SetColVisible(sheetName, "T", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "U", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "V", false); err != nil {
			log.Errorf("unable to remove scorecard columns : %s", err)
		}
	}

	// Remove the validators columns when that is disable
	if r.Flags.DisableValidators {
		if err := f.SetColVisible(sheetName, "W", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
		if err := f.SetColVisible(sheetName, "X", false); err != nil {
			log.Errorf("unable to remove validator columns : %s", err)
		}
	}

	if err := f.AddTable(sheetName, "A5", "AP5", pkg.TableFormat); err != nil {
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
