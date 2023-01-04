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
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/iancoleman/orderedmap"
	"github.com/mpvl/unique"
	"github.com/operator-framework/audit/cmd/index/bundles"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/actions"
	index "github.com/operator-framework/audit/pkg/reports/eus"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

	for index, modelOrDB := range modelOrDBs {
		var EUSReportColumn []channelGrouping
		for _, operator := range allOperators {
			channelGrouping := channelsInIndex(modelOrDB, operator, flags.Indexes[index])
			channelGrouping.MaxOCPPerHead = getMaxOcp(modelOrDB, channelGrouping)
			channelGrouping.Deprecated = getDeprecated(modelOrDB, operator)
			channelGrouping.NonHeadBundles = getNonHeadBundles(modelOrDB, channelGrouping)
			EUSReportColumn = append(EUSReportColumn, channelGrouping)
		}
		EUSReportTable = append(EUSReportTable, EUSReportColumn)
	}
	EUSReportTable = addCommonChannels(EUSReportTable)

	generateJSON(flags.Indexes, EUSReportTable)
	pkg.CleanupTemporaryDirs()
	log.Info("Operation completed.")
	return nil
}

// find intersection of channelGroupings in each row and store to commonChannels field
// needs to handle cases where operator does not exist in a given index, channelNames == nil
func addCommonChannels(table [][]channelGrouping) [][]channelGrouping {
	channelGroupingsByOperatorAcrossIndexes := transpose(table)
	var commonChannels []string
	for _, cgs := range channelGroupingsByOperatorAcrossIndexes {
		foundFirstNonEmptyChannels := false
		last := 1
		for idx, cg := range cgs {
			if cg.ChannelNames != nil && !foundFirstNonEmptyChannels {
				commonChannels = cg.ChannelNames
				foundFirstNonEmptyChannels = true
			}
			if cg.ChannelNames != nil {
				if idx == len(cgs)-1 {
					last = 0
				}
				if cgs[idx+last].ChannelNames == nil {
					commonChannels = cg.ChannelNames
				} else {
					commonChannels = sliceutil.IntersectStrings(commonChannels, cgs[idx+last].ChannelNames)
				}
			}
		}
		// when done, update all the channelGrouping.commonChannels for the operator
		for i := 0; i < len(cgs); i++ {
			if cgs[i].ChannelNames != nil {
				cgs[i].CommonChannels = commonChannels
			}
		}
	}
	// don't reverse the transpose to leave table in row order to match JSON report from older code
	return channelGroupingsByOperatorAcrossIndexes
}

func generateJSON(indexInfo []string, EUSTableData [][]channelGrouping) {
	var reportVersionsSuffix string
	for idx, index := range indexInfo {
		reportVersionsSuffix = reportVersionsSuffix + actions.GetVersionTagFromImage(index)
		if idx < len(indexInfo)-1 {
			reportVersionsSuffix = reportVersionsSuffix + "_"
		} else {
			reportVersionsSuffix = reportVersionsSuffix + ".json"
		}
	}
	JSONReportFile := path.Join("EUS_report_" + reportVersionsSuffix)

	data := make(map[string][]orderedmap.OrderedMap)
	var DataItems []orderedmap.OrderedMap
	//todo see if we can debug hit on Deprecated != nil
	for _, EUSTableRow := range EUSTableData {
		for index, channelGrouping := range EUSTableRow {
			for idx, channelName := range channelGrouping.ChannelNames {
				DataItem := orderedmap.New()
				DataItem.Set("name", channelGrouping.OperatorName)
				DataItem.Set("ocpVersion", actions.GetVersionTagFromImage(indexInfo[index]))
				defaultPostfix := isDefaultChannel(channelName, channelGrouping.DefaultChannelName)
				DataItem.Set("channel", channelName+defaultPostfix)
				maxOCPVersion := ""
				if channelGrouping.MaxOCPPerHead != nil {
					if channelGrouping.MaxOCPPerHead[idx] != "" {
						maxOCPVersion = " (maxOCP = " + channelGrouping.MaxOCPPerHead[idx] + ")"
					}
				}
				DataItem.Set("currentVersion", getVersion(channelGrouping.HeadBundleNames[idx])+maxOCPVersion)
				for idx2, nonHeadBundleName := range channelGrouping.NonHeadBundles[idx] {
					DataItem.Set("otherAvailableVersion"+strconv.Itoa(idx2), getVersion(nonHeadBundleName))
				}
				DataItem.Set("isCommon", checkCommon(channelName, channelGrouping.CommonChannels))
				DataItems = append(DataItems, *DataItem)
			}
		}
	}
	data["data"] = DataItems
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	err = os.WriteFile(JSONReportFile, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func isDefaultChannel(channelName string, nameOfDefaultChannel string) string {
	if channelName == nameOfDefaultChannel {
		return " (default)"
	}
	return ""
}

func checkCommon(name string, commonsChannels []string) string {
	if Contains(commonsChannels, name) {
		return "true"
	}
	return "false"
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
			model, _ = declcfg.ConvertToModel(*fbc)
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

// Determine the channels for operator in an index
func channelsInIndex(modelOrDb interface{}, operator string, ocpIndex string) channelGrouping {
	var channelGrouping channelGrouping
	channelGrouping, err := getChannelsDefaultChannelHeadBundle(modelOrDb, operator)
	if err != nil {
		log.Infof("in index %s: %v (not published in this index?)",
			actions.GetVersionTagFromImage(ocpIndex), err)
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

// for a given operator package in an index store:
// [the channels], [the head bundles for those channels],
// and the default channel
type channelGrouping struct {
	channels           []*model.Channel // not really meant to be used, just a helper for the FBC ones
	OperatorName       string           `json:"name"`
	ChannelNames       []string         `json:"channelName"`
	DefaultChannelName string           `json:"defaultChannelName"`
	HeadBundleNames    []string         `json:"headBundleName"`
	MaxOCPPerHead      []string         `json:"maxOCPPerHead"`
	Deprecated         []string         `json:"deprecated"`
	CommonChannels     []string         `json:"commonChannels"`
	NonHeadBundles     [][]string       `json:"nonHeadBundles"`
}

func getChannelsDefaultChannelHeadBundle(modelOrDb interface{}, operatorName string) (channelGrouping, error) {
	var channelGrouping = channelGrouping{}
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		sql := `SELECT c.name, p.default_channel, c.head_operatorbundle_name  
		FROM package p, channel c 
    	JOIN package on p.name = c.package_name 
		WHERE package_name = ? 
		GROUP BY c.name;`

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
				channelGrouping.OperatorName = operatorName
				channelGrouping.ChannelNames = append(channelGrouping.ChannelNames, channelName)
				channelGrouping.DefaultChannelName = defaultChannelName
				channelGrouping.HeadBundleNames = append(channelGrouping.HeadBundleNames, headBundleName)
			}
		}
		return channelGrouping, nil
	case model.Model:
		for packageName, Package := range modelOrDb {
			if packageName == operatorName {
				channelGrouping.OperatorName = operatorName
				channelGrouping.DefaultChannelName = Package.DefaultChannel.Name
				for _, Channel := range Package.Channels {
					channelGrouping.channels = append(channelGrouping.channels, Channel)
					channelGrouping.ChannelNames = append(channelGrouping.ChannelNames, Channel.Name)
				}
				channelGrouping.DefaultChannelName = Package.DefaultChannel.Name
				for _, channelInPackage := range channelGrouping.channels {
					headBundle, _ := channelInPackage.Head()
					channelGrouping.HeadBundleNames = append(channelGrouping.HeadBundleNames, headBundle.Name)
				}
			}
		}
		return channelGrouping, fmt.Errorf("operator named %q not found in the index", operatorName)
	}
	return channelGrouping, nil
}

func getVersion(bundleName string) string {
	version := strings.Join(strings.Split(bundleName, ".")[1:], ".")
	return version
}

func getNonHeadBundles(modelOrDb interface{}, grouping channelGrouping) [][]string {
	nonHeadBundleNames := make([][]string, len(grouping.ChannelNames))
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		for i, channelName := range grouping.ChannelNames {
			var nonHeadBundleNamesPerChannel []string
			sql := `SELECT operatorbundle.name 
				FROM operatorbundle 
				INNER JOIN channel_entry 
				ON operatorbundle.name=channel_entry.operatorbundle_name 
				WHERE channel_entry.package_name = ? AND channel_entry.channel_name = ?;`
			row, err := modelOrDb.Query(sql, grouping.OperatorName, channelName)
			if err != nil {
				log.Errorf("unable to query the index db for maxOCPs : %s", err)
				return nil
			}
			defer row.Close()
			for row.Next() {
				var bundleName string
				err = row.Scan(&bundleName)
				if err != nil {
					nonHeadBundleNamesPerChannel = append(nonHeadBundleNamesPerChannel, "")
				}
				nonHeadBundleNamesPerChannel = append(nonHeadBundleNamesPerChannel, bundleName)
			}
			nonHeadBundleNames[i] = remove(nonHeadBundleNamesPerChannel, grouping.HeadBundleNames[i])
		}
		return nonHeadBundleNames
	case model.Model:
		for _, Package := range modelOrDb {
			if Package.Name == grouping.OperatorName {
				i := 0
				for _, Channel := range Package.Channels {
					var nonHeadBundleNamesPerChannel []string
					for _, bundle := range Channel.Bundles {
						nonHeadBundleNamesPerChannel = append(nonHeadBundleNamesPerChannel, bundle.Name)
					}
					headBundle, _ := Channel.Head()
					nonHeadBundleNamesPerChannel = remove(nonHeadBundleNamesPerChannel, headBundle.Name)
					nonHeadBundleNames[i] = nonHeadBundleNamesPerChannel
					i++
				}
			}
		}
		return nonHeadBundleNames
	}
	return nonHeadBundleNames
}

func remove(nonHeadBundles []string, headBundle string) []string {
	for i, v := range nonHeadBundles {
		if v == headBundle {
			return append(nonHeadBundles[:i], nonHeadBundles[i+1:]...)
		}
	}
	return nonHeadBundles
}

func getMaxOcp(modelOrDb interface{}, channelGrouping channelGrouping) []string {
	var maxOcpPerChannel []string
	switch modelOrDb := modelOrDb.(type) {
	case *sql.DB:
		for _, channelHead := range channelGrouping.HeadBundleNames {
			sql :=
				"SELECT p.value FROM properties p WHERE p.operatorbundle_name = ? AND type = 'olm.maxOpenShiftVersion';"
			row, err := modelOrDb.Query(sql, channelHead)
			if err != nil {
				log.Errorf("unable to query the index db for maxOCPs : %s", err)
				return nil
			}
			if !row.Next() {
				maxOcpPerChannel = append(maxOcpPerChannel, "")
				continue
			} else {
				var maxOpenShiftVersion string
				err = row.Scan(&maxOpenShiftVersion)
				if err == nil {
					maxOcpPerChannel = append(maxOcpPerChannel, maxOpenShiftVersion)
				}
			}
			row.Close()
		}
		return maxOcpPerChannel
	//TODO debug on 4.11 and verify FBC results are same as SQL here
	case model.Model:
		for _, Package := range modelOrDb {
			if Package.Name == channelGrouping.OperatorName {
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

func transpose(slice [][]channelGrouping) [][]channelGrouping {
	xl := len(slice[0])
	yl := len(slice)
	result := make([][]channelGrouping, xl)
	for i := range result {
		result[i] = make([]channelGrouping, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			result[i][j] = slice[j][i]
		}
	}
	return result
}

func stripQuotes(data []byte) string {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	return string(data)
}
