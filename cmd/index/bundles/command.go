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
	"strings"

	"github.com/operator-framework/audit/pkg/actions"

	"github.com/spf13/cobra"

	// To allow create connection to query the index database
	_ "github.com/mattn/go-sqlite3"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	index "github.com/operator-framework/audit/pkg/reports/bundles"
)

var flags = index.BindFlags{}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundles",
		Short: "audit all operator bundles of an index catalog image",
		Long: "Provides reports with the details of all bundles operators ship in the index image informed " +
			"according to the criteria defined via the flags.\n\n " +
			"**When this report is useful?** \n\n" +
			"This report is useful when is required to check the operator bundles details.",
		PreRunE: validation,
		RunE:    run,
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	cmd.Flags().StringVar(&flags.IndexImage, "index-image", "",
		"index image and tag which will be audit")
	if err := cmd.MarkFlagRequired("index-image"); err != nil {
		log.Fatalf("Failed to mark `index-image` flag for `index` sub-command as required")
	}

	cmd.Flags().StringVar(&flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().StringVar(&flags.OutputFormat, "output", pkg.Xls,
		fmt.Sprintf("inform the output format. [Flags: %s, %s, %s]", pkg.JSON,
			pkg.Xls, pkg.All))
	cmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().Int32Var(&flags.Limit, "limit", 0,
		"limit the num of operator bundles to be audit")
	cmd.Flags().BoolVar(&flags.HeadOnly, "head-only", false,
		"if set, will just check the operator bundle which are head of the channels")
	cmd.Flags().BoolVar(&flags.DisableScorecard, "disable-scorecard", false,
		"if set, will disable the scorecard tests")
	cmd.Flags().BoolVar(&flags.DisableValidators, "disable-validators", false,
		"if set, will disable the validators tests")
	cmd.Flags().StringVar(&flags.Label, "label", "",
		"filter by bundles which has index images where contains *label*")
	cmd.Flags().StringVar(&flags.LabelValue, "label-value", "",
		"filter by bundles which has index images where contains *label=label-value*. "+
			"This option can only be used with the --label flag.")

	return cmd
}

func validation(cmd *cobra.Command, args []string) error {

	if flags.Limit < 0 {
		return fmt.Errorf("invalid value informed via the --limit flag :%v", flags.Limit)
	}

	if len(flags.OutputFormat) > 0 && flags.OutputFormat != pkg.JSON &&
		flags.OutputFormat != pkg.Xls && flags.OutputFormat != pkg.All {
		return fmt.Errorf("invalid value informed via the --output flag :%v. "+
			"The available options are: %s, %s and %s", flags.OutputFormat, pkg.JSON, pkg.Xls, pkg.All)
	}

	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	if len(flags.LabelValue) > 0 && len(flags.Label) < 0 {
		return fmt.Errorf("inform the label via the --label flag")
	}

	if !flags.DisableScorecard {
		if !pkg.HasClusterRunning() {
			return errors.New("this report is configured to run the Scorecard tests which requires a cluster up " +
				"and running. Please, startup your cluster or use the flag --disable-scorecard")
		}
		if !pkg.HasSDKInstalled() {
			return errors.New("this report is configured to run the Scorecard tests which requires the " +
				"SDK CLI version >= 1.5 installed locally.\n" +
				"Please, see ensure that you have SDK installed or use the flag --disable-scorecard.\n" +
				"More info: https://github.com/operator-framework/operator-sdk")
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")

	reportData := index.Data{}
	reportData.Flags = flags
	pkg.GenerateTemporaryDirs()

	// to fix common possible typo issue
	reportData.Flags.Filter = strings.ReplaceAll(reportData.Flags.Filter, "”", "")

	if err := actions.DownloadImage(flags.IndexImage); err != nil {
		return err
	}

	// Inspect the OLM index image
	var err error
	reportData.IndexImageInspect, err = pkg.RunDockerInspect(flags.IndexImage)
	if err != nil {
		log.Errorf("unable to inspect the index image: %s", err)
	}

	if err := actions.ExtractIndexDB(flags.IndexImage); err != nil {
		return err
	}

	reportData, err = getDataFromIndexDB(reportData)
	if err != nil {
		return err
	}

	log.Infof("Start to generate the reportData")
	if err := reportData.OutputReport(); err != nil {
		return err
	}

	pkg.CleanupTemporaryDirs()
	log.Infof("Operation completed.")

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
		var csv *string
		var bundlePath string
		var skipRange string
		var version string
		var replaces string
		var skips string
		var csvStruct *v1alpha1.ClusterServiceVersion

		err = row.Scan(&bundleName, &csv, &bundlePath, &version, &skipRange, &replaces, &skips)
		if err != nil {
			log.Errorf("unable to scan data from index %s\n", err.Error())
		}

		auditBundle := models.NewAuditBundle(bundleName, bundlePath)

		// the csv is pruned from the database to save space.
		// See that is store only what is needed to populate the package manifest on cluster, all the extra
		// manifests are pruned to save storage space
		if csv != nil {
			err = json.Unmarshal([]byte(*csv), &csvStruct)
			if err == nil {
				auditBundle.CSVFromIndexDB = csvStruct
			} else {
				auditBundle.Errors = append(auditBundle.Errors,
					fmt.Errorf("unable to parse the csv from the index.db: %s", err).Error())
			}
		}

		auditBundle.VersionDB = version
		auditBundle.SkipRangeDB = skipRange
		auditBundle.ReplacesDB = replaces
		auditBundle.SkipsDB = skips

		auditBundle = actions.GetDataFromBundleImage(auditBundle, report.Flags.DisableScorecard,
			report.Flags.DisableValidators, report.Flags.Label, report.Flags.LabelValue)

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
			auditBundle.Channels = append(auditBundle.Channels, channelName)
			auditBundle.PackageName = packageName
		}

		if len(strings.TrimSpace(auditBundle.PackageName)) == 0 && auditBundle.Bundle != nil {
			auditBundle.PackageName = auditBundle.Bundle.Package
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

		sqlString = fmt.Sprintf("SELECT type, value FROM properties WHERE operatorbundle_name = '%s'",
			auditBundle.OperatorBundleName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query properties entry in the index db : %s", err)
		}

		defer row.Close()
		var properType string
		var properValue string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&properType, &properValue)
			auditBundle.PropertiesDB = append(auditBundle.PropertiesDB,
				pkg.PropertiesAnnotation{Type: properType, Value: properValue})
		}

		sqlString = fmt.Sprintf("select count(*) from channel where head_operatorbundle_name = '%s'",
			auditBundle.OperatorBundleName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query properties entry in the index db : %s", err)
		}

		defer row.Close()
		var found int
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&found)
			auditBundle.IsHeadOfChannel = found > 0
		}

		report.AuditBundle = append(report.AuditBundle, *auditBundle)
	}

	return report, nil
}
