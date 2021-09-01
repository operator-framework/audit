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

// Deprecated
// This script is only a helper for we are able to know what are the bundles that we need to
// deprecated on 4.9. That will be removed as soon as possible and is just added
// here in case it be required to be checked and used so far.
// The following script uses the JSON format output image to
// generates the output.yml file which has all packages which still
// are without a head of channel compatible with 4.9.
// The idea is provide a helper to allow to send emails to notify their authors
// Example of usage: (see that we leave makefile target to help you out here)
// nolint: lll
// go run hack/scripts/ivs-emails/generate.go --mongo=mongo-query-join-results-prod.json --image=testdata/reports/redhat_certified_operator_index/bundles_registry.redhat.io_redhat_certified_operator_index_v4.8_2021-08-10.json
// go run hack/scripts/ivs-emails/generate.go --mongo=mongo-query-join-results-prod.json --image=testdata/reports/redhat_redhat_marketplace_index/bundles_registry.redhat.io_redhat_redhat_marketplace_index_v4.8_2021-08-06.json
// todo: remove after 4.9-GA
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

type MongoContacts struct {
	Type  string `json:"type"`
	Email string `json:"email_address"`
}

type CertProject struct {
	Contacts []MongoContacts `json:"contacts,omitempty"`
}

type MongoItems struct {
	Association string        `json:"association"`
	PackageName string        `json:"package_name"`
	CertProject []CertProject `json:"cert_project,omitempty"`
}

type Item struct {
	MongoItem          MongoItems
	CSVEmails          []string
	CSVLinks           []string
	Warnings           map[string]string
	SuggestedEmailBody string
}

type ImageData struct {
	ImageName   string
	ImageID     string
	ImageHash   string
	ImageBuild  string
	GeneratedAt string
}

type Output struct {
	Items     []Item
	ImageData ImageData
}

//nolint: lll
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var mongoFile string
	var jsonFile string

	flag.StringVar(&mongoFile, "mongo", "", "Inform the path for the mongo file with the reqqured data to generate the file. ")
	flag.StringVar(&jsonFile, "image", "", "Inform the path for the JSON result which will be used to generate the report. ")

	flag.Parse()

	byteValue, err := pkg.ReadFile(filepath.Join(currentPath, mongoFile))
	if err != nil {
		log.Fatal(err)
	}

	var mongoValues []MongoItems
	if err = json.Unmarshal(byteValue, &mongoValues); err != nil {
		log.Fatal(err)
	}
	var result Output

	items, image := getData(filepath.Join(currentPath, jsonFile), mongoValues)
	result.Items = items
	result.ImageData = image

	reportPath := filepath.Join(currentPath, hack.ReportsPath, "ivs")
	command := exec.Command("mkdir", reportPath)
	_, _ = pkg.RunCommand(command)

	fp := filepath.Join(reportPath, pkg.GetReportName(result.ImageData.ImageName, "ivs", "json"))
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = hack.ReplaceInFile(fp, "", string(jsonResult))
	if err != nil {
		log.Fatal(err)
	}

}

func getData(image string, mongoValues []MongoItems) ([]Item, ImageData) {
	apiDashReport, err := getAPIDashForImage(image)
	if err != nil {
		log.Fatal(err)
	}

	var items []Item
	for _, pkgV := range apiDashReport.PartialComplying {
		mg := MongoItems{PackageName: pkgV.Name, Association: "N/A"}
		for _, m := range mongoValues {
			if m.PackageName == pkgV.Name {
				mg = m
				break
			}
		}
		var emails []string
		var links []string
		warns := make(map[string]string, len(pkgV.AllBundles))
		for _, v := range pkgV.AllBundles {

			emails = append(emails, v.MaintainersEmail...)
			links = append(links, v.Links...)
			if len(v.BundleName) > 0 && v.ValidatorWarnings != nil {
				for _, w := range v.ValidatorWarnings {
					if strings.Contains(w, "1.22") {
						warns[v.BundleName] = strings.ReplaceAll(w, "this bundle", "this distribution")
					}
				}
			}
		}
		emails = pkg.GetUniqueValues(emails)
		links = pkg.GetUniqueValues(links)

		warn := ""
		for _, w := range pkgV.AllBundles {
			if len(warns[w.BundleName]) > 0 {
				warn += fmt.Sprintf("- **%s**: %s \n", w.BundleName, warns[w.BundleName])
			}
		}

		body := strings.ReplaceAll(bodyMsg, "<warnings>", warn)
		body = strings.ReplaceAll(body, "<name>", pkgV.Name)

		items = append(items, Item{MongoItem: mg, CSVEmails: emails, CSVLinks: links, Warnings: warns, SuggestedEmailBody: body})
	}

	sort.Slice(items[:], func(i, j int) bool {
		return items[i].MongoItem.PackageName < items[j].MongoItem.PackageName
	})

	var imageData ImageData

	imageData.ImageName = apiDashReport.ImageName
	imageData.ImageBuild = apiDashReport.ImageBuild
	imageData.ImageID = apiDashReport.ImageID
	imageData.GeneratedAt = apiDashReport.GeneratedAt

	return items, imageData
}

func getAPIDashForImage(image string) (*custom.APIDashReport, error) {
	// Update here the path of the JSON report for the image that you would like to be used
	custom.Flags.File = image

	bundlesReport, err := custom.ParseBundlesJSONReport()
	if err != nil {
		log.Fatal(err)
	}

	apiDashReport := custom.NewAPIDashReport(bundlesReport)
	return apiDashReport, err
}

const bodyMsg = `
Dear maintainer,

Kubernetes has been deprecating API(s), which will be removed and are no longer available in 1.22. Operators projects using these APIs[0] versions will **not** work on Kubernetes 1.22 or any cluster vendor using this Kubernetes version(1.22), such as OpenShift 4.9+. Following the APIs that are most likely your projects to be affected by:
- apiextensions.k8s.io/v1beta1: (Used for CRDs and available since v1.16)
- rbac.authorization.k8s.io/v1beta1: (Used for RBAC/rules and available since v1.8)
- admissionregistration.k8s.io/v1beta1 (Used for Webhooks and available since v1.16)

Therefore, looks like this project distributes solutions via the Red Hat Connect[1] with the package name as <name> and does not contain any version compatible with k8s 1.22/OCP 4.9. Following some findings by checking the distributions published:

<warnings>

**NOTE:** The above findings are only about the manifests shipped inside of the distribution. It is not checking the codebase.

### How to solve

It would be very nice to see new distributions of this project that are no longer using these APIs[0] and so they can work on Kubernetes 1.22 and newer and be published in the Red Hat Connect[1] collection. OpenShift 4.9, for example, will not ship operators anymore that do still use v1beta1 extension APIs.

Due to the number of options available to build Operators, it is hard to provide direct guidance on updating your operator to support Kubernetes 1.22. Recent versions of the OperatorSDK[2] greater than 1.0.0[3] and Kubebuilder[4] greater than 3.0.0[5] scaffold your project with the latest versions of these APIs (all that is generated by tools only). See the guides to upgrade your projects with OperatorSDK: 

- Golang - https://sdk.operatorframework.io/docs/building-operators/golang/migration/ 
- Ansible - https://sdk.operatorframework.io/docs/building-operators/ansible/migration/
- Helm - https://sdk.operatorframework.io/docs/building-operators/helm/migration/ 

Also, see the guide to upgrade your projects with Kubebuilder:

- Kubebuilder - https://book.kubebuilder.io/migration/v2vsv3.html. 

For APIs other than the ones mentioned above, you will have to check your code for usage of removed API versions and upgrade to newer APIs. The details of this depend on your codebase.

**If this projects only need to migrate the API for CRDs and it was built with OperatorSDK[2] versions lower than 1.0.0[3] then, you maybe able to solve it with an OperatorSDK[2] version >= v0.18.x < 1.0.0:**

> $ operator-sdk generate crds --crd-version=v1
> INFO[0000] Running CRD generator.                      
> INFO[0000] CRD generation complete.

**Alternatively, you can try to upgrade your manifests with controller-gen[6] version >= v0.4.1[7]:**

#### If this project does not use Webhooks:

> $ controller-gen crd:trivialVersions=true,preserveUnknownFields=false rbac:roleName=manager-role  paths="./..."

#### If this project is using Webhooks:

1. Add the markers sideEffects[8] and admissionReviewVersions[9] to your webhook (Example with sideEffects=None and admissionReviewVersions={v1,v1beta1}: memcached-operator/api/v1alpha1/memcached_webhook.go[10]):

2. Run the command:
> $ controller-gen crd:trivialVersions=true,preserveUnknownFields=false rbac:roleName=manager-role webhook paths="./..."

For further info and tips see the blog: https://connect.redhat.com/blog/api-deprecation-kubernetes-122-will-impact-your-operators.

Thank you for your attention.

[0] - https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22
[1] - https://connect.redhat.com/
[2] - https://github.com/operator-framework/operator-sdk
[3] - https://github.com/operator-framework/operator-sdk/releases
[4] - https://github.com/kubernetes-sigs/kubebuilder
[5] - https://github.com/kubernetes-sigs/kubebuilder/releases
[6] - https://book.kubebuilder.io/reference/controller-gen.html
[7] - https://github.com/kubernetes-sigs/controller-tools/releases/tag/v0.4.1
[8] - https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/webhook/parser.go#L81
[9] - https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/webhook/parser.go#L114
[10] - https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v3/memcached-operator/api/v1alpha1/memcached_webhook.go#L39
`
