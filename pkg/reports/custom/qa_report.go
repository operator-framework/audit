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
)

// nolint:golint
const DEPRECATED_API_NOT_COMPLY = "NOT COMPLY"

// nolint:golint
const DEPRECATED_API_COMPLY = "COMPLY"

// nolint:golint
const PASS = "PASSED IN ALL CHECKS"

// nolint:golint
const WARNINGS = "CHECK THE WARNINGS"

// nolint:golint
const FOUND = "FOUND"

// nolint:golint
const USED = "USED"

// nolint:golint
const NOT_USED = "NOT USED"

// nolint:golint
const ERRORS_WARNINGS = "FIX ERRORS AND WARNINGS"

// nolint:golint
const ERRORS = "ONLY ERRORS"

const sdkBuilderAnnotation = "operators.operatorframework.io/builder"

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

type PackageQA struct {
	PackageName                 string
	DeprecateAPI                []string
	DeprecateAPIColor           string
	CapabilityColor             string
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
	ChannelNamesNotComply       []string
	ChannelNamesComply          []string
	BundlesWithoutDisconnect    []string
	HeadOfChannels              []BundleDeprecate
	Capabilities                []string
	Subscriptions               []string
}

type QAReport struct {
	ImageName    string
	ImageID      string
	ImageHash    string
	ImageBuild   string
	GeneratedAt  string
	PackageGrade []PackageQA
}

func NewQAReport(bundlesReport bundles.Report, filter string) *QAReport {
	gradeReport := QAReport{}
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
		bd.AddPotentialWarning()
		allBundles = append(allBundles, bd)
	}

	mapPackagesWithBundles := MapBundlesPerPackage(allBundles)

	isRedHatIndex := false
	if strings.Contains(gradeReport.ImageName, "redhat-operator-index") {
		isRedHatIndex = true
	}
	for key, bds := range mapPackagesWithBundles {
		if len(key) == 0 {
			continue
		}
		if len(bds) == 0 {
			continue
		}
		pkgGrade := NewPkg(key, bds, isRedHatIndex)
		gradeReport.PackageGrade = append(gradeReport.PackageGrade, pkgGrade)
	}

	return &gradeReport
}

func NewPkg(pkgName string, bundlesOfPkg []BundleDeprecate, isRedHatIndex bool) PackageQA {

	pkg := PackageQA{PackageName: pkgName}

	pkg.CapabilityColor = BLACK
	pkg.DisconnectedAnnotationColor = BLACK
	pkg.ChannelNamingColor = BLACK
	pkg.SDKUsageColor = BLACK
	pkg.ScorecardDefaultImagesColor = BLACK
	pkg.ScorecardCustomImagesColor = BLACK
	pkg.ValidatorsColor = BLACK

	pkg.HeadOfChannels = GetHeadOfChannels(bundlesOfPkg)

	pkg.checkCapability()
	pkg.checkDisconnectAnnotation(isRedHatIndex)
	pkg.checkScorecard()
	pkg.checkValidators()
	pkg.checkChannelNamingScore()
	pkg.checkSDKUsage()
	pkg.checkScorecardCustom()
	pkg.checkRemovalAPIs1_25_26()
	pkg.checkSubscriptions()

	return pkg
}

func (p *PackageQA) checkCapability() {

	var levels []string
	for _, v := range p.HeadOfChannels {

		l := v.BundleData.BundleCSV.Annotations["capabilities"]
		switch l {
		case "Basic Install":
			p.CapabilityColor = ORANGE
		case "Seamless Upgrades":
			p.CapabilityColor = ORANGE
		case "Full Lifecycle":
			p.CapabilityColor = GREEN
		case "Deep Insights":
			p.CapabilityColor = GREEN
		case "Auto Pilot":
			p.CapabilityColor = GREEN
		default:
			l = l + " - (Invalid level value)"
			p.CapabilityColor = RED
		}
		levels = append(levels, l)
	}

	p.Capabilities = pkg.GetUniqueValues(levels)
}

func (p *PackageQA) checkSDKUsage() {
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
	} else {
		p.SDKUsageColor = BLACK
		p.SDKUsage = NOT_USED
	}
}

func (p *PackageQA) checkScorecardCustom() {
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
	} else {
		p.ScorecardCustomImagesColor = BLACK
		p.ScorecardCustomImages = NOT_USED
	}
}

func (p *PackageQA) checkRemovalAPIs1_25_26() {

	var listOfWarnings []string
	for _, v := range p.HeadOfChannels {
		listOfWarnings = append(listOfWarnings, v.Permissions1_25...)
		listOfWarnings = append(listOfWarnings, v.Permissions1_26...)
	}

	listOfWarnings = pkg.GetUniqueValues(listOfWarnings)

	if len(listOfWarnings) > 0 {
		p.DeprecateAPIColor = ORANGE
		p.DeprecateAPI = listOfWarnings
	} else {
		p.DeprecateAPIColor = GREEN
	}
}

func (p *PackageQA) checkChannelNamingScore() {
	var foundErrors []string
	var OK []string
	for _, v := range p.HeadOfChannels {
		for _, c := range v.BundleData.Channels {
			if !pkg.IsFollowingChannelNameConventional(c) {
				foundErrors = append(foundErrors, c)
			} else {
				OK = append(OK, c)
			}
		}
	}

	if len(foundErrors) > 0 {
		p.ChannelNamingColor = YELLOW
		p.ChannelNamesNotComply = pkg.GetUniqueValues(foundErrors)
		p.ChannelNaming = "NOT COMPLY"
	} else {
		p.ChannelNamingColor = GREEN
		p.ChannelNaming = "PROBABLY COMPLY"
	}
	p.ChannelNamesComply = pkg.GetUniqueValues(OK)
}

func (p *PackageQA) checkScorecard() {
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
	} else if !foundErrors && foundWarnings {
		p.ScorecardDefaultImages = WARNINGS
		p.ScorecardDefaultImagesColor = YELLOW
	} else if foundErrors && foundWarnings {
		p.ScorecardDefaultImagesColor = RED
		p.ScorecardDefaultImages = ERRORS_WARNINGS
	} else if foundErrors && !foundWarnings {
		p.ScorecardDefaultImagesColor = RED
		p.ScorecardDefaultImages = ERRORS
	}
}

func (p *PackageQA) checkValidators() {
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
	} else if !foundErrors && foundWarnings {
		p.ValidatorsColor = YELLOW
		p.Validators = WARNINGS
	} else if foundErrors && foundWarnings {
		p.ValidatorsColor = RED
		p.Validators = ERRORS_WARNINGS
	} else if foundErrors && !foundWarnings {
		p.ValidatorsColor = RED
		p.Validators = ERRORS
	}
}

func (p *PackageQA) checkDisconnectAnnotation(isRedHatIndex bool) {
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
	} else {
		if found {
			p.DisconnectedAnnotation = "REQUIRED"
			p.DisconnectedAnnotationColor = RED
		} else {
			if isRedHatIndex {
				p.DisconnectedAnnotation = "SHOULD SUPPORT"
				p.DisconnectedAnnotationColor = ORANGE
			} else {
				p.DisconnectedAnnotation = "NOT USED"
				p.DisconnectedAnnotationColor = GREEN
			}
		}
	}
}

func (p *PackageQA) checkSubscriptions() {
	const subscription = "operators.openshift.io/valid-subscription"

	// Check if found subscription on the CSV
	for _, bundle := range p.HeadOfChannels {
		if bundle.BundleData.BundleCSV != nil && len(bundle.BundleData.BundleCSV.Annotations) > 0 &&
			len(bundle.BundleData.BundleCSV.Annotations[subscription]) > 0 {

			// If is not the default channel then, ignore the scenario
			if !p.isDefaultChannel(bundle) {
				continue
			}

			value := bundle.BundleData.BundleCSV.Annotations[subscription]
			list := strings.Split(value, ",")
			for _, v := range list {
				v = strings.ReplaceAll(v, "[", "")
				v = strings.ReplaceAll(v, "]", "")
				p.Subscriptions = append(p.Subscriptions, v)
			}
		}
	}

	//Check if found subscription on the annotations file (metadata/annotations.yaml)
	for _, bundle := range p.HeadOfChannels {
		if bundle.BundleData.BundleAnnotations != nil && len(bundle.BundleData.BundleAnnotations) > 0 &&
			len(bundle.BundleData.BundleAnnotations[subscription]) > 0 {

			// If is not the default channel then, ignore the scenario
			if !p.isDefaultChannel(bundle) {
				continue
			}

			value := bundle.BundleData.BundleAnnotations[subscription]
			list := strings.Split(value, ",")

			for _, v := range list {
				v = strings.ReplaceAll(v, "[", "")
				v = strings.ReplaceAll(v, "]", "")
				p.Subscriptions = append(p.Subscriptions, v)
			}
		}
	}

	// Some teams adds in both so it is to ensure that we have not duplicated results
	p.Subscriptions = pkg.GetUniqueValues(p.Subscriptions)
}

func (p *PackageQA) isDefaultChannel(bundle BundleDeprecate) bool {
	for _, channel := range bundle.BundleData.Channels {
		if bundle.BundleData.DefaultChannel == channel {
			return true
		}
	}
	return false
}
