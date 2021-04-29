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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	// To allow create connection to query the index database
	_ "github.com/mattn/go-sqlite3"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/actions"
	"github.com/operator-framework/audit/pkg/models"
	index "github.com/operator-framework/audit/pkg/reports/bundles"
)

const catalogIndex = "audit-catalog-index"

var flags = index.BindFlags{}

func NewBundlesCmd() *cobra.Command {
	bundlesCmd := &cobra.Command{
		Use:   "bundles",
		Short: "audit all operator bundles of an index catalog image",
		Long: "Provides reports with the details of all bundles operators ship in the index image informed " +
			"according to the criteria defined via the flags.\n\n " +
			"**When this report is useful?** \n\n" +
			"This report is useful when is required to check the operator bundles details.",
		PreRunE: validation,
		RunE:    indexRun,
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	bundlesCmd.Flags().StringVar(&flags.IndexImage, "index-image", "",
		"index image and tag which will be audit")
	if err := bundlesCmd.MarkFlagRequired("index-image"); err != nil {
		log.Fatalf("Failed to mark `index-image` flag for `index` sub-command as required")
	}

	bundlesCmd.Flags().StringVar(&flags.Filter, "filter", "",
		"filter by operator bundle names which are *filter*")
	bundlesCmd.Flags().StringVar(&flags.OutputFormat, "output", pkg.Xls,
		"inform the output format. [Flags: xls, json]. (Default: xls)")
	bundlesCmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	bundlesCmd.Flags().Int32Var(&flags.Limit, "limit", 0,
		"limit the num of operator bundles to be audit")
	bundlesCmd.Flags().BoolVar(&flags.HeadOnly, "head-only", false,
		"if set, will just check the operator bundle which are head of the channels")
	bundlesCmd.Flags().BoolVar(&flags.DisableScorecard, "disable-scorecard", false,
		"if set, will disable the scorecard tests")
	bundlesCmd.Flags().BoolVar(&flags.DisableValidators, "disable-validators", false,
		"if set, will disable the validators tests")
	bundlesCmd.Flags().StringVar(&flags.Label, "label", "",
		"filter by bundles which has index images where contains *label*")
	bundlesCmd.Flags().StringVar(&flags.LabelValue, "label-value", "",
		"filter by bundles which has index images where contains *label=label-value*. "+
			"This option can only be used with the --label flag.")

	return bundlesCmd
}

func validation(cmd *cobra.Command, args []string) error {

	if flags.Limit < 0 {
		return fmt.Errorf("invalid value informed via the --limit flag :%v", flags.Limit)
	}

	if len(flags.OutputFormat) > 0 && flags.OutputFormat != pkg.JSON && flags.OutputFormat != pkg.Xls {
		return fmt.Errorf("invalid value informed via the --output flag :%v. "+
			"The available options are %s,%s", flags.Limit, pkg.JSON, pkg.Xls)
	}

	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	if len(flags.LabelValue) > 0 && len(flags.Label) < 0 {
		return fmt.Errorf("inform the label via the --label flag")
	}

	return nil
}

func indexRun(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")
	reportData := index.Data{}
	reportData.Flags = flags

	// Create tmp dir to process the reportData
	// Cleanup
	command := exec.Command("rm", "-rf", "tmp")
	_, _ = pkg.RunCommand(command)
	command = exec.Command("mkdir", "tmp")
	_, err := pkg.RunCommand(command)
	if err != nil {
		return err
	}

	if err := extractIndexDB(); err != nil {
		return err
	}

	reportData, err = getDataFromIndexDB(reportData)
	if err != nil {
		return err
	}

	// Cleanup
	command = exec.Command("rm", "-rf", "tmp")
	_, _ = pkg.RunCommand(command)

	log.Infof("Start to generate the reportData")
	if err := reportData.OutputReport(); err != nil {
		return err
	}
	log.Infof("Operation completed.")

	return nil
}

func extractIndexDB() error {
	// Remove image if exists already
	command := exec.Command("docker", "rm", catalogIndex)
	_, _ = pkg.RunCommand(command)

	// Download the image
	command = exec.Command("docker", "create", "--name", catalogIndex, flags.IndexImage, "\"yes\"")
	_, err := pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to create container image %s : %s", flags.IndexImage, err)
	}

	// Extract
	command = exec.Command("docker", "cp", fmt.Sprintf("%s:/database/index.db", catalogIndex), "./output/")
	_, err = pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to extract the image for index.db %s : %s", flags.IndexImage, err)
	}
	return nil
}

func getDataFromIndexDB(report index.Data) (index.Data, error) {
	// Connect to the database
	db, err := sql.Open("sqlite3", "./output/index.db")
	if err != nil {
		return report, fmt.Errorf("unable to connect in to the database : %s", err)
	}

	sql, err := report.BuildBundlesQuery()
	if err != nil {
		return report, err
	}

	row, err := db.Query(sql)
	if err != nil {
		return report, fmt.Errorf("unable to query the index db : %s", err)
	}

	defer row.Close()
	for row.Next() {
		var bundleName string
		var csv string
		var bundlePath string
		var skipRange string
		var version string
		var replaces string
		var skips string
		var csvStruct *v1alpha1.ClusterServiceVersion

		_ = row.Scan(&bundleName, &csv, &bundlePath, &version, &skipRange, &replaces, &skips)

		auditBundle := models.NewAuditBundle(bundleName, bundlePath)
		err = json.Unmarshal([]byte(csv), &csvStruct)
		if err == nil {
			auditBundle.CSVFromIndexDB = csvStruct
		} else {
			auditBundle.Errors = append(auditBundle.Errors, fmt.Errorf("not found csv stored or" +
				" unable to unmarshal data from the index.db: %s", err))
		}

		auditBundle.VersionDB = version
		auditBundle.SkipRangeDB = skipRange
		auditBundle.ReplacesDB = replaces
		auditBundle.SkipsDB = skips

		if len(bundlePath) > 0 {
			// todo: improve the labels filter by implementing it in another way
			auditBundle = actions.GetDataFromBundleImage(auditBundle, report.Flags.DisableScorecard,
				report.Flags.DisableValidators, report.Flags.Label, report.Flags.LabelValue)
		} else {
			auditBundle.Errors = append(auditBundle.Errors,
				errors.New("not found bundle path stored in the index.db"))
		}

		sqlString := fmt.Sprintf("SELECT c.channel_name, c.package_name FROM channel_entry c "+
			"where c.operatorbundle_name = '%s'", auditBundle.OperatorBundleName)
		row, err := db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query channel entry in the index db : %s", err)
		}

		defer row.Close()
		var channelName string
		var packageName string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&channelName, &packageName)
			auditBundle.BundleChannel = channelName
			auditBundle.PackageName = packageName
		}

		sqlString = fmt.Sprintf("SELECT default_channel FROM package WHERE name = '%s'", auditBundle.PackageName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query default channel entry in the index db : %s", err)
		}

		defer row.Close()
		var defaultChannelName string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&defaultChannelName)
			auditBundle.DefaultChannel = defaultChannelName
		}

		report.AuditBundle = append(report.AuditBundle, *auditBundle)
	}

	return report, nil
}
