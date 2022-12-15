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

package eus

import (
	"database/sql"
	"fmt"
	"github.com/mpvl/unique"
	"github.com/operator-framework/audit/cmd/index/bundles"
	"github.com/operator-framework/audit/pkg/actions"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"sort"

	"github.com/operator-framework/audit/pkg"
	index "github.com/operator-framework/audit/pkg/reports/eus"
)

var flags = index.BindFlags{}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eus",
		Short: "generate an EUS Report",
		Long: `Create a report of possible upgrade paths (bundle versions, channels) from one OCP EUS version to another 

## When should I use it?

This command generate an EUS Report.
By running this command audit tool will:

- Gather information from one OCP EUS version index to another and all the indexes in between
- Output a report providing the information obtained and processed in JSON format.

`,

		PreRunE: validation,
		RunE:    run,
	}

	cmd.Flags().StringSliceVarP(&flags.Indexes, "indexes", "", []string{},
		"Red Hat EUS index version number for report \"from\" (inclusive))")
	if err := cmd.MarkFlagRequired("indexes"); err != nil {
		log.Fatalf("Failed to set `indexes` flag with list of indexes for `eus` sub-command as required")
	}

	return cmd
}

func validation(cmd *cobra.Command, args []string) error {

	if len(flags.OutputFormat) > 0 && flags.OutputFormat != pkg.JSON {
		return fmt.Errorf("invalid value informed via the --output flag :%v. "+
			"The available option is: %s", flags.OutputFormat, pkg.JSON)
	}

	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	if len(flags.ContainerEngine) == 0 {
		flags.ContainerEngine = pkg.GetContainerToolFromEnvVar()
	}
	if flags.ContainerEngine != pkg.Docker && flags.ContainerEngine != pkg.Podman {
		return fmt.Errorf("invalid value for the flag --container-engine (%s)."+
			" The valid options are %s and %s", flags.ContainerEngine, pkg.Docker, pkg.Podman)
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")

	pkg.GenerateTemporaryDirs()

	// sorted list of operators, each once, that appear in any of the indexes:
	var allOperators []string
	var EUSReportTable [][]channelGrouping
	modelOrDBs := getModelsOrDB(flags.Indexes)

	// get all the operators in all the indexes in the range
	for _, modelOrDB := range modelOrDBs {
		allOperatorsPerIndex, err := getPackageNames(modelOrDB)
		if err == nil {
			allOperators = append(allOperators, allOperatorsPerIndex...)
		}
	}
	sort.Strings(allOperators)
	unique.Strings(&allOperators)

	for _, modelOrDB := range modelOrDBs {
		var EUSReportColumn []channelGrouping
		for _, operator := range allOperators {
			channelGrouping := channelsAcrossIndexes(modelOrDB, operator)
			channelGrouping.maxOCPPerHead = getMaxOcp(modelOrDB, operator)

			EUSReportColumn = append(EUSReportColumn, channelGrouping)
		}
		EUSReportTable = append(EUSReportTable, EUSReportColumn)
	}

	pkg.CleanupTemporaryDirs()
	log.Info("Operation completed.")
	return nil
}

func getModelsOrDB(indexes []string) []any {
	var modelsOrDBs []any
	for _, index := range indexes {
		if err := actions.ExtractIndexDBorCatalogs(index, flags.ContainerEngine); err != nil {
			log.Errorf("error on passed indexes: %s", err)
			return modelsOrDBs
		}
		log.Infof("Preparing Data for EUS Report for index %s...", index)

		var model model.Model
		var db *sql.DB
		var modelOrDB any
		var err error

		if bundles.IsFBC(index) {
			// newer file-based catalogs
			root := "./output/" + actions.GetVersionTagFromImage(index) + "/configs"
			fileSystem := os.DirFS(root)
			fbc, err := declcfg.LoadFS(fileSystem)

			if err != nil {
				log.Errorf("unable to load the file based config : %s", err)
				return modelsOrDBs
			}
			model, err = declcfg.ConvertToModel(*fbc)
		} else {
			// older sqlite index
			db, err = sql.Open("sqlite3", "./output/"+
				actions.GetVersionTagFromImage(index)+"/index.db")
			if err != nil {
				return modelsOrDBs
			}
		}
		if model != nil {
			modelOrDB = model
		} else {
			modelOrDB = db
		}
		modelsOrDBs = append(modelsOrDBs, modelOrDB)
	}
	return modelsOrDBs
}

//// builds a list of operators that exist in all the indexes
//func allOperatorsExist(modelOrDb interface{}, allOperators []string) []string {
//	var existingOperators []string
//	for _, operator := range allOperators {
//		if isOperatorInIndex(modelOrDb, operator) {
//			existingOperators = append(existingOperators, operator)
//		}
//	}
//	return existingOperators
//}

// Determine common channels that exist across all indexes
func channelsAcrossIndexes(modelOrDb interface{}, operator string) channelGrouping {
	var channelGrouping channelGrouping
	channelGrouping, err := getChannelsDefaultChannelHeadBundle(modelOrDb, operator)
	if err != nil {
		log.Errorf("error finding channel info for %s in index %s: %v", operator, modelOrDb, err)
	}
	return channelGrouping
}

func getPackageNames(modelOrDb interface{}) ([]string, error) {
	var packageNames []string
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		sql := "SELECT p.name FROM package p;"

		row, err := modelOrDb.Query(sql)
		if err != nil {
			return nil, fmt.Errorf("unable to query the index db : %s", err)
		}
		defer row.Close()
		for row.Next() {
			var packageName string
			err = row.Scan(&packageName)
			if err != nil {
				log.Errorf("unable to scan data from index %s\n", err.Error())
			} else {
				packageNames = append(packageNames, packageName)
			}
		}
		return packageNames, nil
	case model.Model:
		for _, Package := range modelOrDb {
			packageNames = append(packageNames, Package.Name)
		}
		return packageNames, nil
	}
	return nil, nil
}

//func isOperatorInIndex(modelOrDb interface{}, operatorName string) bool {
//	switch modelOrDb := modelOrDb.(type) {
//	case *sql.DB:
//		var packageNames []string
//		sql := "SELECT p.name FROM package p WHERE name = ?;"
//
//		row, err := modelOrDb.Query(sql, operatorName)
//		if err != nil {
//			log.Errorf("unable to query the index db : %s", err)
//			return false
//		}
//		defer row.Close()
//		for row.Next() {
//			var packageName string
//			err = row.Scan(&packageName)
//			if err != nil {
//				log.Errorf("unable to scan data from index %s\n", err.Error())
//			} else {
//				packageNames = append(packageNames, packageName)
//			}
//		}
//		return len(packageNames) != 0
//	case model.Model:
//		packagesNames, err := getPackageNames(modelOrDb)
//		if err == nil {
//			if Contains(packagesNames, operatorName) {
//				return true
//			}
//		}
//		return false
//	}
//	return false
//}

// for a given operator package in an index store:
// [the channels], [the head bundles for those channels],
// and the default channel
type channelGrouping struct {
	channels           []*model.Channel // not really meant to be used, just a helper for the FBC ones
	operatorName       string
	channelNames       []string
	defaultChannelName string
	headBundleNames    []string
	maxOCPPerHead      []string
}

func getChannelsDefaultChannelHeadBundle(modelOrDb interface{}, operatorName string) (channelGrouping, error) {
	var channelGrouping = channelGrouping{}
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		sql := "SELECT c.name, p.default_channel, c.head_operatorbundle_name" +
			"    FROM package p, channel c " +
			"    JOIN package on p.name = c.package_name" +
			"    WHERE package_name = ? " +
			"    GROUP BY c.name;"

		row, err := modelOrDb.Query(sql, operatorName)
		if err != nil {
			return channelGrouping, fmt.Errorf("unable to query the index db : %s", err)
		}
		defer row.Close()
		for row.Next() {
			var channelName string
			var defaultChannelName string
			var headBundleName string
			err = row.Scan(&channelName, &defaultChannelName, &headBundleName)
			if err == nil {
				channelGrouping.operatorName = operatorName
				channelGrouping.channelNames = append(channelGrouping.channelNames, channelName)
				channelGrouping.defaultChannelName = defaultChannelName
				channelGrouping.headBundleNames = append(channelGrouping.headBundleNames, headBundleName)
			}
		}
		return channelGrouping, nil
	case model.Model:
		for packageName, Package := range modelOrDb {
			if packageName == operatorName {
				for _, Channel := range Package.Channels {
					channelGrouping.channels = append(channelGrouping.channels, Channel)
					channelGrouping.channelNames = append(channelGrouping.channelNames, Channel.Name)
				}
				channelGrouping.defaultChannelName = Package.DefaultChannel.Name
				for _, channelInPackage := range channelGrouping.channels {
					headBundle, _ := channelInPackage.Head()
					channelGrouping.headBundleNames = append(channelGrouping.headBundleNames, headBundle.Name)
				}
			}
		}
		return channelGrouping, fmt.Errorf("operator named %q not found in the index", operatorName)
	}
	return channelGrouping, nil
}

func getMaxOcp(modelOrDb interface{}, operatorName string) []string {
	var maxOcpPerChannel []string
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		sql := "SELECT p.value FROM properties p WHERE p.operatorbundle_name = ? AND type = \"olm.maxOpenShiftVersion\""
		row, err := modelOrDb.Query(sql, operatorName)
		if err != nil {
			log.Errorf("unable to query the index db : %s", err)
			return nil
		}
		defer row.Close()
		for row.Next() {
			var maxOpenShiftVersion string
			err = row.Scan(&maxOpenShiftVersion)
			if err == nil {
				maxOcpPerChannel = append(maxOcpPerChannel, maxOpenShiftVersion)
			}
		}
		return maxOcpPerChannel
	case model.Model:
		for _, Package := range modelOrDb {
			if Package.Name == operatorName {
				for _, Channel := range Package.Channels {
					headBundle, _ := Channel.Head()
					for _, Bundle := range Channel.Bundles {
						for _, property := range Bundle.Properties {
							if property.Type == "olm.maxOpenShiftVersion" && Bundle.Name == headBundle.Name {
								maxOcpPerChannel = append(maxOcpPerChannel, stripQuotes(property.Value))
							}
						}
					}
				}
			}
		}
		return maxOcpPerChannel
	}
	return maxOcpPerChannel
}

func getDeprecated(modelOrDb interface{}, operatorName string) []string {
	var deprecates []string
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		sql := "SELECT d.operatorbundle_name FROM deprecated d WHERE d.operatorbundle_name = ?;"
		row, err := modelOrDb.Query(sql, operatorName)
		if err != nil {
			log.Errorf("unable to query the index db : %s", err)
			return nil
		}
		defer row.Close()
		for row.Next() {
			var deprecated string
			err = row.Scan(&deprecated)
			if err == nil {
				deprecates = append(deprecates, deprecated)
			}
		}
		return deprecates
	case model.Model:
		for _, Package := range modelOrDb {
			for _, Channel := range Package.Channels {
				for _, Bundle := range Channel.Bundles {
					for _, property := range Bundle.Properties {
						if property.Type == "olm.deprecated" {
							deprecates = append(deprecates, Bundle.Name)
						}
					}
				}
			}
		}
		return deprecates
	}
	return deprecates
}

func Contains[T comparable](arr []T, x T) bool {
	for _, v := range arr {
		if v == x {
			return true
		}
	}
	return false
}

func stripQuotes(data []byte) string {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	return string(data)
}
