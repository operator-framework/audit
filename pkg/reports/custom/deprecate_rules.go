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
	"fmt"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/audit/pkg/reports/bundles"
)

type BundleDeprecate struct {
	BundleData        bundles.Column
	DeprecateAPIsMsgs []string
	ApisRemoved1_22   []string
	ApisRemoved1_25   []string
	ApisRemoved1_26   []string
	Permissions1_25   []string
	Permissions1_26   []string
}

// (Green) Complying
// If is not using deprecated API(s) at all in the head channels
// If has at least one channel head which is compatible with 4.9 (migrated)
// and the other head channels are with max ocp version
func MapPkgsComplyingWithDeprecateAPI(
	mapPackagesWithBundles map[string][]BundleDeprecate, k8sVersion string) map[string][]BundleDeprecate {
	complying := make(map[string][]BundleDeprecate)
	for key, bundlesPerPkg := range mapPackagesWithBundles {
		// has bundlesPerPkg that we cannot find the package
		// some inconsistency in the index db.
		// So, this scenario can only be added to the complying if all is migrated
		if key == "" {
			if !hasNotMigratedAPIFor(bundlesPerPkg, k8sVersion) {
				complying[key] = mapPackagesWithBundles[key]
			}
			continue
		}

		if hasHeadOfChannelMigratedAPIFor(bundlesPerPkg, k8sVersion) {
			complying[key] = mapPackagesWithBundles[key]
		}
	}
	return complying
}

func hasNotMigratedAPIFor(bundlesPerPkg []BundleDeprecate, k8sVersion string) bool {
	foundNotMigrated := false
	for _, v := range bundlesPerPkg {
		switch k8sVersion {
		case "1.26":
			if len(v.ApisRemoved1_26) > 0 && !v.BundleData.IsDeprecated {
				foundNotMigrated = true
				break
			}
		case "1.25":
			if len(v.ApisRemoved1_25) > 0 && !v.BundleData.IsDeprecated {
				foundNotMigrated = true
				break
			}
		default:
			if len(v.ApisRemoved1_22) > 0 && !v.BundleData.IsDeprecated {
				foundNotMigrated = true
				break
			}
		}
	}
	return foundNotMigrated
}

func hasHeadOfChannelMigratedAPIFor(bundlesPerPkg []BundleDeprecate, k8sversion string) bool {
	for _, v := range bundlesPerPkg {
		switch k8sversion {
		case k8s126:
			if (v.ApisRemoved1_26 == nil || len(v.ApisRemoved1_26) < 1) &&
				v.BundleData.IsHeadOfChannel && !v.BundleData.IsDeprecated {
				return true
			}
		case k8s125:
			if (v.ApisRemoved1_25 == nil || len(v.ApisRemoved1_25) < 1) &&
				v.BundleData.IsHeadOfChannel && !v.BundleData.IsDeprecated {
				return true
			}
		default:
			if (v.ApisRemoved1_22 == nil || len(v.ApisRemoved1_22) < 1) &&
				v.BundleData.IsHeadOfChannel && !v.BundleData.IsDeprecated {
				return true
			}
		}
	}
	return false
}

func (bd *BundleDeprecate) AddDeprecateDataFromValidators() {
	for _, result := range bd.BundleData.ValidatorErrors {
		bd.setDeprecateMsg(result)
	}
	for _, result := range bd.BundleData.ValidatorWarnings {
		bd.setDeprecateMsg(result)
	}
}

func (bd *BundleDeprecate) AddPotentialWarning() {

	if bd == nil || bd.BundleData.BundleCSV == nil {
		return
	}

	// We need looking for clusterPermissions and permissions
	apis125 := map[string][]string{
		"batch":            {"cronjobs"},
		"discovery.k8s.io": {"endpointslices"},
		"events.k8s.io":    {"events"},
		"autoscaling":      {"horizontalpodautoscalers"},
		"policy":           {"poddisruptionbudgets", "podsecuritypolicies"},
		"node.k8s.io":      {"runtimeclasses"},
	}

	apis126 := map[string][]string{
		"flowcontrol.apiserver.k8s.io": {"flowschemas", "prioritylevelconfigurations"},
		"autoscaling":                  {"horizontalpodautoscalers"},
	}

	for _, perm := range bd.BundleData.BundleCSV.Spec.InstallStrategy.StrategySpec.Permissions {
		bd.addFromRules(perm, apis125, apis126)
	}

	for _, perm := range bd.BundleData.BundleCSV.Spec.InstallStrategy.StrategySpec.ClusterPermissions {
		bd.addFromRules(perm, apis125, apis126)
	}
}

func (bd *BundleDeprecate) addFromRules(perm v1alpha1.StrategyDeploymentPermissions,
	apis125 map[string][]string, apis126 map[string][]string) {
	if perm.Rules == nil {
		return
	}
	for apiFromMap, resourcesFromMap := range apis125 {
		for _, rule := range perm.Rules {
			for _, api := range rule.APIGroups {
				if strings.EqualFold(api, apiFromMap) {
					for _, res := range rule.Resources {
						for _, resFromMap := range resourcesFromMap {
							if strings.EqualFold(resFromMap, res) {
								bd.Permissions1_25 = append(bd.Permissions1_25,
									fmt.Sprintf("(apiGroups/resources): %s/%s", api, res))
							}
							if strings.ToLower(res) == "*" || strings.ToLower(res) == "[*]" {
								bd.Permissions1_25 = append(bd.Permissions1_25,
									fmt.Sprintf("(All from apiGroups): %s/%s", api, res))
							}
						}
					}
				}
			}
		}
	}

	for apiFromMap, resourcesFromMap := range apis126 {
		for _, rule := range perm.Rules {
			for _, api := range rule.APIGroups {
				if strings.EqualFold(api, apiFromMap) {
					for _, res := range rule.Resources {
						for _, resFromMap := range resourcesFromMap {
							if strings.EqualFold(resFromMap, res) {
								bd.Permissions1_26 = append(bd.Permissions1_26,
									fmt.Sprintf("(apiGroups/resources): %s/%s", api, res))
							}
							if strings.ToLower(res) == "*" || strings.ToLower(res) == "[*]" {
								bd.Permissions1_26 = append(bd.Permissions1_26,
									fmt.Sprintf("(All from apiGroups): %s/%s", api, res))
							}
						}
					}
				}
			}
		}
	}
}

func (bd *BundleDeprecate) setDeprecateMsg(result string) {
	if strings.Contains(result, "this bundle is using APIs which were deprecated") {
		bd.DeprecateAPIsMsgs = append(bd.DeprecateAPIsMsgs, result)
		if strings.Contains(result, "1.22") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_22, result)
		}
		if strings.Contains(result, "1.25") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_25, result)
		}
		if strings.Contains(result, "1.26") {
			bd.ApisRemoved1_22 = append(bd.ApisRemoved1_26, result)
		}
	}
}
