// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this File except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package custom

import (
	"strings"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	log "github.com/sirupsen/logrus"
)

// nolint:golint
const DEPRECATED_API_NOT_COMPLY = "NOT COMPLY"

// nolint:golint
const DEPRECATED_API_COMPLY = "COMPLY"

// nolint:golint
const PASS = "PASS"

// nolint:golint
const WARNINGS = "ONLY WARNINGS"

// nolint:golint
const FOUND = "FOUND"

// nolint:golint
const USED = "USED"

// nolint:golint
const NOT_USED = "NOT USED"

// nolint:golint
const ERRORS_WARNINGS = "ERRORS AND WARNINGS"

// nolint:golint
const ERRORS = "ONLY ERRORS"

const sdkBuilderAnnotation = "operators.operatorframework.io/builder"

// DeprecateAPI max total : 400
// Scorecard and Validators max total for each: 200 - PASS or WARNINGS
// Scorecard checks can bring extra 100+ (optional for A+) - if FOUND custom tests
// DisconnectedAnnotation and SDK USAGE max total : 100
// total : 1100
var scoreMap = map[string]int{
	DEPRECATED_API_COMPLY:     400,
	DEPRECATED_API_NOT_COMPLY: 0,
	PASS:                      200,
	WARNINGS:                  100,
	FOUND:                     100,
	USED:                      100,
	NOT_USED:                  0,
}

const RED = "red"
const YELLOW = "#ec8f1c"
const GREEN = "green"
const ORANGE = "orange"
const BLACK = "black"

//the following data came from: https://access.redhat.com/articles/4740011
var pkgsThatSupportsDisconnectedMode = []string{
	"3scale apicast-operator",
	"3scale-operator",
	"amq-streams",
	"businessautomation-operator",
	"cam-operator",
	"cluster-logging",
	"codeready-toolchain-operator",
	"codeready-workspaces",
	"compliance-operator",
	"datagrid",
	"elasticsearch-operator",
	"file-integrity-operator",
	"fuse-online",
	"jaeger-product",
	"jenkins-operator",
	"kiali-ossm",
	"kubevirt-hyperconverge",
	"local-storage-operator",
	"metering-ocp",
	"mtc-operator",
	"nfd",
	"ocs-operator",
	"openshift-gitops-operator",
	"openshift", // seems that all that we have now should support //todo: identify all names
	"ptp-operator",
	"quay", // todo: check the specific names either
	"serverless-operator",
	"servicemeshoperator",
	"sriov-network-operator",
}

type PackageGrade struct {
	PackageName                 string
	DeprecateAPI                string
	DeprecateAPIColor           string
	DisconnectedAnnotation      string
	DisconnectedAnnotationColor string
	ChannelNaming               string
	ChannelNamingColor          string
	SDKUsage                    string
	SDKUsageColor               string
	ScorecardDefaultImages      string
	ScorecardDefaultImagesColor string
	ScorecardCustomImages       string
	ScorecardCustomImagesColor  string
	Validators                  string
	ValidatorsColor             string
	Score                       int
	Grade                       string
	ChannelNamesNotComply       []string
	BundlesWithoutDisconnect    []string
	HeadOfChannels              []BundleDeprecate
}

type GradeReport struct {
	ImageName    string
	ImageID      string
	ImageHash    string
	ImageBuild   string
	GeneratedAt  string
	PackageGrade []PackageGrade
}

func NewGradeReport(bundlesReport bundles.Report, filter string) *GradeReport {
	gradeReport := GradeReport{}
	gradeReport.ImageName = bundlesReport.Flags.IndexImage
	gradeReport.ImageID = bundlesReport.IndexImageInspect.ID
	gradeReport.ImageBuild = bundlesReport.IndexImageInspect.DockerConfig.Labels["build-date"]
	gradeReport.GeneratedAt = bundlesReport.GenerateAt

	var allBundles []BundleDeprecate
	for _, v := range bundlesReport.Columns {
		// filter by the name
		if len(filter) > 0 {
			if !strings.Contains(v.PackageName, filter) {
				continue
			}
		}
		bd := BundleDeprecate{BundleData: v}
		bd.AddDeprecateDataFromValidators()
		allBundles = append(allBundles, bd)
	}

	mapPackagesWithBundles := MapBundlesPerPackage(allBundles)
	//todo: fix to check 1.25 and 1.26 as well
	migrated := MapPkgsComplyingWithDeprecateAPI(mapPackagesWithBundles, "1.22")
	notMigrated := make(map[string][]BundleDeprecate)
	for key := range mapPackagesWithBundles {
		if len(migrated[key]) == 0 {
			notMigrated[key] = mapPackagesWithBundles[key]
		}
	}

	for key, bds := range mapPackagesWithBundles {
		if len(key) == 0 {
			continue
		}
		pkgGrade := NewPkgGrade(key, bds, notMigrated, migrated)
		gradeReport.PackageGrade = append(gradeReport.PackageGrade, pkgGrade)
	}

	return &gradeReport
}

func NewPkgGrade(pkgName string, bundlesOfPkg []BundleDeprecate,
	notMigrated, migrated map[string][]BundleDeprecate) PackageGrade {

	pkgGrade := PackageGrade{PackageName: pkgName}

	pkgGrade.DeprecateAPIColor = BLACK
	pkgGrade.DisconnectedAnnotationColor = BLACK
	pkgGrade.ChannelNamingColor = BLACK
	pkgGrade.SDKUsageColor = BLACK
	pkgGrade.ScorecardDefaultImagesColor = BLACK
	pkgGrade.ScorecardCustomImagesColor = BLACK
	pkgGrade.ValidatorsColor = BLACK

	pkgGrade.HeadOfChannels = GetHeadOfChannels(bundlesOfPkg)

	pkgGrade.checkDeprecatedAPIScore(notMigrated, migrated)
	pkgGrade.checkDisconnectAnnotationScore()
	pkgGrade.checkScorecardScore()
	pkgGrade.checkValidatorsScore()
	pkgGrade.checkChannelNamingScore()
	pkgGrade.checkSDKUsageScore()
	pkgGrade.checkScorecardCustom()

	if pkgGrade.Score < 400 {
		pkgGrade.Grade = "Grade D"
	} else if pkgGrade.Score >= 400 && pkgGrade.Score < 600 {
		pkgGrade.Grade = "Grade C"
	} else if pkgGrade.Score >= 600 && pkgGrade.Score < 900 {
		pkgGrade.Grade = "Grade B"
	} else if pkgGrade.Score >= 900 {
		pkgGrade.Grade = "Grade A"
	}
	return pkgGrade
}

func (p *PackageGrade) checkDeprecatedAPIScore(notMigrated map[string][]BundleDeprecate,
	migrated map[string][]BundleDeprecate) {
	if notMigrated[p.PackageName] != nil {
		p.DeprecateAPI = DEPRECATED_API_NOT_COMPLY
		p.DeprecateAPIColor = YELLOW
		p.Score += scoreMap[DEPRECATED_API_NOT_COMPLY]
	} else if migrated[p.PackageName] != nil {
		p.DeprecateAPI = DEPRECATED_API_COMPLY
		p.DeprecateAPIColor = GREEN
		p.Score += scoreMap[DEPRECATED_API_COMPLY]
	} else {
		log.Errorf("unable to check the deprecated API score for the pkg %s", p.PackageName)
	}
}

func (p *PackageGrade) checkSDKUsageScore() {
	found := false
	for _, v := range p.HeadOfChannels {
		builder := v.BundleData.BundleCSV.Annotations[sdkBuilderAnnotation]
		if len(builder) < 1 {
			builder = v.BundleData.BundleAnnotations[sdkBuilderAnnotation]
		}
		if strings.Contains(builder, "operator-sdk") {
			found = true
			break
		}
	}

	if found {
		p.SDKUsageColor = GREEN
		p.SDKUsage = USED
		p.Score += scoreMap[USED]
	} else {
		p.SDKUsageColor = BLACK
		p.SDKUsage = NOT_USED
	}
}

func (p *PackageGrade) checkScorecardCustom() {
	found := false
	for _, v := range p.HeadOfChannels {
		if v.BundleData.HasCustomScorecardTests {
			found = true
			break
		}
	}

	if found {
		p.ScorecardCustomImagesColor = GREEN
		p.ScorecardCustomImages = USED
		p.Score += scoreMap[USED]
	} else {
		p.ScorecardCustomImagesColor = BLACK
		p.ScorecardCustomImages = NOT_USED
	}
}

func (p *PackageGrade) checkChannelNamingScore() {
	var foundErrors []string
	for _, v := range p.HeadOfChannels {
		for _, c := range v.BundleData.Channels {
			if !pkg.IsFollowingChannelNameConventional(c) {
				foundErrors = append(foundErrors, c)
			}
		}
	}

	if len(foundErrors) > 0 {
		p.ChannelNamingColor = YELLOW
		p.ChannelNamesNotComply = pkg.GetUniqueValues(foundErrors)
		p.ChannelNaming = "NOT COMPLY"
	} else {
		p.ChannelNamingColor = GREEN
		p.ChannelNaming = "COMPLY"
		p.Score += 100
	}
}

func (p *PackageGrade) checkScorecardScore() {
	foundErrors := false
	foundWarnings := false
	for _, v := range p.HeadOfChannels {
		if len(v.BundleData.ScorecardErrors) > 0 {
			foundErrors = true
		}

		if len(v.BundleData.ScorecardSuggestions) > 0 {
			foundWarnings = true
		}
	}

	if !foundErrors && !foundWarnings {
		p.ScorecardDefaultImages = PASS
		p.ScorecardDefaultImagesColor = GREEN
		p.Score += scoreMap[PASS]
	} else if !foundErrors && foundWarnings {
		p.ScorecardDefaultImages = WARNINGS
		p.ScorecardDefaultImagesColor = YELLOW
		p.Score += scoreMap[WARNINGS]
	} else if foundErrors && foundWarnings {
		p.ScorecardDefaultImagesColor = ORANGE
		p.ScorecardDefaultImages = ERRORS_WARNINGS
	} else if foundErrors && !foundWarnings {
		p.ScorecardDefaultImagesColor = ORANGE
		p.ScorecardDefaultImages = ERRORS
	}
}

func (p *PackageGrade) checkValidatorsScore() {
	foundErrors := false
	foundWarnings := false
	for _, v := range p.HeadOfChannels {
		if len(v.BundleData.ValidatorErrors) > 0 {
			foundErrors = true
		}

		if len(v.BundleData.ValidatorWarnings) > 0 {
			foundWarnings = true
		}
	}

	if !foundErrors && !foundWarnings {
		p.ValidatorsColor = GREEN
		p.Validators = PASS
		p.Score += scoreMap[PASS]
	} else if !foundErrors && foundWarnings {
		p.ValidatorsColor = YELLOW
		p.Validators = WARNINGS
		p.Score += scoreMap[WARNINGS]
	} else if foundErrors && foundWarnings {
		p.ValidatorsColor = ORANGE
		p.Validators = ERRORS_WARNINGS
	} else if foundErrors && !foundWarnings {
		p.ValidatorsColor = ORANGE
		p.Validators = ERRORS
	}
}

func (p *PackageGrade) checkDisconnectAnnotationScore() {
	found := false
	for _, v := range pkgsThatSupportsDisconnectedMode {
		if strings.Contains(p.PackageName, v) {
			found = true
			break
		}
	}
	for _, b := range p.HeadOfChannels {
		infra := b.BundleData.BundleCSV.ObjectMeta.Annotations[pkg.InfrastructureAnnotation]
		if !strings.Contains(infra, "Disconnected") && !strings.Contains(infra, "disconnected") {
			p.BundlesWithoutDisconnect = append(p.BundlesWithoutDisconnect, b.BundleData.BundleCSV.Name)
		}
	}

	if len(p.BundlesWithoutDisconnect) == 0 {
		p.DisconnectedAnnotation = USED
		p.DisconnectedAnnotationColor = GREEN
		p.Score += scoreMap[USED]
	} else {
		if found {
			p.DisconnectedAnnotation = "REQUIRED"
			p.DisconnectedAnnotationColor = RED
		} else {
			p.DisconnectedAnnotation = "NOT USED"
			p.DisconnectedAnnotationColor = GREEN
		}
	}
}
