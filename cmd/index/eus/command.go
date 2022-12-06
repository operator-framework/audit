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
	"github.com/operator-framework/audit/cmd/index/bundles"
	"github.com/operator-framework/audit/pkg/actions"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"

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

	//TODO needs to change to working w/ already existing local indexes (registry login & download handled out of band)
	for _, index := range flags.Indexes {
		if err := actions.ExtractIndexDBorCatalogs(index, flags.ContainerEngine); err != nil {
			return err
		}
	}
	log.Info("Preparing Data for EUS Report...")
	var err error

	// check here to see if it's index.db or file-based catalogs
	if bundles.IsFBC() {
		root := "./output/configs"
		fileSystem := os.DirFS(root)
		fbc, err := declcfg.LoadFS(fileSystem)

		if err != nil {
			return fmt.Errorf("unable to load the file based config : %s", err)
		}
		model, err := declcfg.ConvertToModel(*fbc)
		//getMaxOcpFBC(model, "cluster-logging")
		deprecates := getDeprecatedFBC(model)
		print(deprecates)
	}
	if err != nil {
		return err
	}

	pkg.CleanupTemporaryDirs()
	log.Info("Operation completed.")

	return nil
}

type Package = *model.Package

// pass db from something like: sql.Open("sqlite3", "./output/"+index+"/index.db")
func getPackageNames(db *sql.DB) ([]string, error) {
	var packageNames []string
	sql := "SELECT p.name FROM package p;"

	row, err := db.Query(sql)
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
}

func getPackageNamesFBC(model model.Model) ([]string, error) {
	var packageNames []string
	for _, Package := range model {
		packageNames = append(packageNames, Package.Name)
	}
	return packageNames, nil
}

func isOperatorInIndex(db *sql.DB, operatorName string) bool {
	var packageNames []string
	sql := "SELECT p.name FROM package p WHERE name = ?;"

	row, err := db.Query(sql, operatorName)
	if err != nil {
		log.Errorf("unable to query the index db : %s", err)
		return false
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
	return len(packageNames) != 0
}

func isOperatorInIndexFBC(model model.Model, operatorName string) bool {
	packagesNames, err := getPackageNamesFBC(model)
	if err == nil {
		if Contains(packagesNames, operatorName) {
			return true
		}
	}
	return false
}

// for a given operator package in an index store:
// [the channels], [the head bundles for those channels],
// and the default channel
type channelGrouping struct {
	channels           []*model.Channel // not really meant to be used, just a helper for the FBC ones
	channelNames       []string
	defaultChannelName string
	headBundleNames    []string
}

func getChannelsDefaultChannelHeadBundle(db *sql.DB, operatorName string) (channelGrouping, error) {
	var channelGrouping = channelGrouping{}
	sql := "SELECT c.name, p.default_channel, c.head_operatorbundle_name" +
		"    FROM package p, channel c " +
		"    JOIN package on p.name = c.package_name" +
		"    WHERE package_name = ? " +
		"    GROUP BY c.name;"

	row, err := db.Query(sql, operatorName)
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
			channelGrouping.channelNames = append(channelGrouping.channelNames, channelName)
			channelGrouping.defaultChannelName = defaultChannelName
			channelGrouping.headBundleNames = append(channelGrouping.headBundleNames, headBundleName)
		}
	}
	return channelGrouping, nil
}

func getChannelsDefaultChannelHeadBundleFBC(model model.Model, operatorName string) (channelGrouping, error) {
	var channelGrouping = channelGrouping{}
	for packageName, Package := range model {
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

func getMaxOcp(db *sql.DB, operatorName string) []string {
	var maxOcpPerChannel []string
	sql := "SELECT p.value FROM properties p WHERE p.operatorbundle_name = ? AND type = \"olm.maxOpenShiftVersion\""
	row, err := db.Query(sql, operatorName)
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
}

func getMaxOcpFBC(model model.Model, operatorName string) []string {
	var maxOcpPerChannel []string
	for _, Package := range model {
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

func getDeprecated(db *sql.DB, operatorName string) []string {
	var deprecates []string
	sql := "SELECT d.operatorbundle_name FROM deprecated d WHERE d.operatorbundle_name = ?;"
	row, err := db.Query(sql, operatorName)
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
}

func getDeprecatedFBC(model model.Model) []string {
	var deprecates []string
	for _, Package := range model {
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
