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
// This script is only a helper for we are able to compare to JSON files
// and check what packages were defined in one and are no longer in the other
// one. E.g After send a test to iib I want to know what packages in green
// were in the JSON A which are no longer in its result JSON B
// todo: remove after 4.9-GA

package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg/reports/bundles"
	"github.com/operator-framework/audit/pkg/reports/custom"
)

//nolint:lll
func main() {

	var deprecated = []string{
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:280bf802843fab435a2946a49324900223f14ae01abbb113624bc056b5ab0188",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:1e1cd6a2950349e3e619e2ffe15cd10d36a5b15e1a9e3b0024bdb11a2355b7ca",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:ea42543f1127fd6ec53cf7f6c7f61f3e0b62f1b210844584d89d60c4bf53fef9",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:75c32ced4add3c6c583daf0bcf81b13cbeb99fd2e24f2c3261fb95ac7bcec4f1",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:c068e6f697764fe0aa82f5572450bbf6cc80d9670a7cb65a8753afa9fa9f8bf6",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:1af785b35226525dc1265123bc274bccb025439ea396e560d0a562791f2c8217",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:3a8c3261d61779907a9a86bb4a2699bd8c60f3ec6466cf6656034b7eda127bc6",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:d97ae02a20eaaa1a0770710a7b5808787d9617ce63c03d3c3f08a11f5bf41aa3",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:e77e15271554a64192f4c80a864efe4f082caeeb94d72f10165beb5cba043ebf",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:c86057b81e3f4a2979f6d5e2ddcbb9fc31549f8d528497a4924e53b26fd75c8c",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:a0a2be3279d03955acf2c2471384c819d45ee765401242eeac20751fe3c835fd",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:76435cfe5728bbcacabb1a444ca45df913a7d5a8541b0cc40496cd11d77865db",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:f02648187cb8cc1c34a6c463ce7f307cab7016da4c06e836971b9bd23d967394",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:9725b91409219faf9e37bab93529ca21bfd209aab3395403866275a3bd08f072",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:510b81899ac25c4318e31a7a44b5c2645fc14e24cc463847f19c5229e8d87c6f",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:5d6c8f7a046eaebd4cee3ad10151bfa3bb176df8e20cd475db66becdcd7356fd",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:4e4fc6cdfec03a175ada82a98044ba6ae80e5d7fc096a24bc5f2977f49d6c70b",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:55906425c2619315d98a9f2b3b014379bfa14875592d18a550a93344097c5496",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:96e323e923dfb556daadc87e990eb89277c21926e56e378bae60bf1d2a46931a",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:adba2e6b585bde1c05d041097a9f19b0753230480032aa44ad1b27fecbf819f5",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:82f8508111f307c25e458a2283e86a21eaeeaa2cd0b335766174f324f6bd392d",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:3560cfe5aa98787496ad1db0440c32a53c9f91ec6bf56fe674b44fcce0913fbc",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:5af9a80acccef3f230dca67538276c3777568cbc98b20d6018c5a9843ab18005",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:2dc9b26f20e89ab7faa9b6a4a4cdf3137061a5115447d0342de268b8fbdd34ad",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:5f9235a715e8091889aff5427683c6a529d222a8126a883da570fb531cb2771b",
		"registry.redhat.io/rhacm2/acm-operator-bundle@sha256:1292fa5177f41a96698a37a35a96d61c1e9c85934a71b4fe91ed9416e78658b3",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:ed1658893b5e78ea6d684dac7f237a80eb1a5fe146996c39152c077ea59f0d61",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:6195b018194b457eb6bad2c53381bdfa4d84329e86d561580a63a9c2faab66d9",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:28157b46dcee1e5fffef9260aa9639df03cca2d8e84ccd63bf1fd966c17c10d5",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:f0f899b90e0ab0e9cc9818e4181beccf178543388c14570da0fd8dda0b4abaa9",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:b350796e748ac56aeaf71711f4eaae06d2b9f77fc1c9aeec1680c9ab096e663c",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:609f34c93dfc53fb853c45d395fcb5c07b6d96e7ab075b036c785203415cfc11",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:81fabc8e6bdc05a772fbf4401331e8ccc568593dafada38fff4c8aaf0565ba06",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:e81aac14ba253b4a89370f8aabfebf491f34621cf8d54efd6b8b3796ca8732bb",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:c5b5ba7265c5ac8e77ff9836746a8ac9ca5e1967270adffffdfe32d8a14968cf",
		"registry.redhat.io/amq7/amq-online-1-controller-manager-rhel7-operator-metadata@sha256:debf79ef45e0fdd229bdfae29ff9a40926854278b39b4aa0d364f5ae9c02c6dc",
		"registry.redhat.io/amq7/amqstreams-rhel7-operator-metadata@sha256:cc19289ff777153ca9ab3cebc45e4c98e9c083437d5e2b6407244aead6e89312",
		"registry.redhat.io/amq7/amqstreams-rhel7-operator-metadata@sha256:6ef3bf83ee075cff73ecc2df897d747b6d589df423dcb3ac1f493d1535e627eb",
		"registry.redhat.io/amq7/amqstreams-rhel7-operator-metadata@sha256:ec906a4abe8e72844e0fd0ebf0abdc312c0b599e22dc3b7c80b9543cdc76b622",
		"registry.redhat.io/amq7/amq-streams-rhel7-operator-metadata@sha256:4ae366be201075ebd86d5bf352d5c77cfa13b8faf5eee5093bfd480ca3a392a2",
		"registry.redhat.io/amq7/amq-streams-rhel7-operator-metadata@sha256:e07c00d943b6951f66c2b629f1086fc4357746b09f647856d8802c4fcb4105fc",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:df348c6ad999c3be37ab19b3651b5a61ab1fa55f7b2ff02b1fea48bfad69ab4a",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:a9d8ced4deb7113de7c97efc226cb9d2ee912be2ce6e67418ddcee423f2a9204",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:c119d2dc15f7d0038623fc2d682e513bba0cbbc63e0095ab769bd119faf59b7f",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:5929b5d30f2c3c215bf0c3052101b618d1e16c39f8b525a22ceb3541009c1a2c",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:3e7da2092cffe5bd2161c98b86e26ce6107248ca52c94a3a333b180f623333de",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:8bd9ac4b1ddcc03f2e6beb131bcfaf0f65ba6b90039e7a1af9dda631ae8d37f8",
		"registry.redhat.io/rhpam-7/rhpam-operator-bundle@sha256:5a5429d84cc36ce29d95fe86d9b6a9eaef0c3ed3d1925fd9b653904c411cbcb4",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:7b7558c8d9170bd27c9dcf2e62df827d477dbbea694f7040f5cd5bcd0eb598a4",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:af5795b85faa63523353eba59b244b94578eed6ee11d0a4f834e64009395a069",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:aba490f87a1ed4c4a47cb5cb5a82ea1bd67031db0382df400d81e13a125f4aff",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:b9bc3d8fb71df47ef32e4f2c7d74b938887fbe2683b132d76e0215eb6cee4a9a",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:1007e3de27d2235e319c23b08dce4ad83d3569f37fff37ad8174b29909e01f04",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:289f77b62c012b6ef09930acf1466749327ef9a1594eb39e077ff7116f5be73b",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:c48e4c3f85e30ab3a209369f7fd9212d34fbe2b97de5334439b06742e9cdec2a",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:181dc432a5aa228ca57a4c23218391f846284e258f72ed306f15ef884e991ee8",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:496aeecec52b0645cfcaa8b7ee7e71c55f823c7520ce3d2c9238d522c17a6e98",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:357f830568a690ea194acdf2b1c5d66b54b10bf1a0dc58d1e771454692dc2270",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:69761d7600d9602fd89f600d9670156e71f68c82802578b0db729cfa0087c89b",
		"registry.redhat.io/codeready-workspaces/crw-2-rhel8-operator-metadata@sha256:080bc58804ca40380c5ed58a5bbdd5fe035fed84a07e58d1c590e06695bbb720",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:7dd45b30f4ffc76846fb6841317c91d6c414b78e0ce10d64f8673447ac2a9066",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:7765bf223baf3e6fd3f5db725586888874280812f017bb01c0bd7714a8bab336",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:8a958713ec2b1f484a688b9dbab0158c6753d59b37dc41075c86dabe6c4eea56",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:7c133eb164050d4e13ff1edbf30394d34b985c313a8b2a333d6ac54e05b66889",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:14fd60e0b98aa6eceec076f5ca340610ff3f9e61a70ff6ebff3da3cc7f855514",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:eee656a31b8ef822fca8c25cc4d5efe11a07090d32410b3a016dd9204fa79247",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:c503cc820237d2aaaf4fa42145e05b3d49645485d343f0c0fa87baa6242a9fd1",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:cc0bb4989fe0e7107d07bcec39e5595291152d602652aa9b6d2a3b7fb24f86c4",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:ec6fd022c1d740c86ee449b1c29817bcf8a7002f62e8c95167b961f6a74a1730",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:37e5af4cf83db8ab63e6e6ddd39aa06100d2fc4110705e88ac9afb3a7abc15c4",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:17009de2d5a855a4570defc332378ce042630784bd29006740d34c435af9008c",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:fe608b121e62e77c39fc6abeb13d1618f2d593811f962e8e3ce477d927c821f1",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:b35b91095373da362968febdfe8d977930d4046e6cd3798960544ada025a665a",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:305c3e345af6adb5ad16c27035c7d4dd42f878dc566184946ce8e403bc83729b",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:52a65c00fed258ec9443c561f71acfdafc1891410451e7ea5a0ea55b72248430",
		"registry.redhat.io/datagrid/datagrid-8-prod-operator-bundle@sha256:517964e827e483f7bd2af4ed60bd23356f4a5d296b1401a14fac03ba7ec8619b",
		"registry.redhat.io/fuse7/fuse-console-operator-bundle@sha256:5e3e9d565510c1a12351f89f912e21f318ee0d7ed52fc8cca6051a6dbf3a6e6d",
		"registry.redhat.io/fuse7/fuse-console-rhel7-operator-metadata@sha256:d207bf3721bdfb6d4d90cda5245b3c49cedef0ba2a308ed64cd00aa6fea7b152",
		"registry.redhat.io/fuse7/fuse-online-operator-bundle@sha256:93d12b2fc2a2e0263890529e227dcb9e37172139f63a95d39b1dce4f19cee39c",
		"registry.redhat.io/fuse7/fuse-online-operator-bundle@sha256:0161c67a3c6be92306f68c2a3b7d9ce3edcf57fdeb1bf2843f988cc85fefc89b",
		"registry.redhat.io/distributed-tracing/jaeger-operator-bundle@sha256:57ce6dfb43fc6d39024f56b1f9d5a3e05be91960d83c19e7a7a43f0af34c3442",
		"registry.redhat.io/distributed-tracing/jaeger-operator-bundle@sha256:ff3ab762bd6d0d9f2cdbe44abf2c0ca39c248e8588e6d4fdf56e49b50bb98d08",
		"registry.redhat.io/distributed-tracing/jaeger-operator-bundle@sha256:1760d894216cd496083f3ccf5bdb44f313d04ef9cedff601b637e3f078cf92a9",
		"registry.redhat.io/distributed-tracing/jaeger-operator-bundle@sha256:2d3b46a96af8b2455bb614ee6db92516a6d9d94e31fe6076acb6e9524c1cb25b",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:58b7976358bd5b30958602c820c5d3b0ba136e65fd7424ce92d23801dbd0976f",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:dd102ee9ec3a32312e63309e5cd5526e8eb1eb7e7b5ee299b95302618c00a0cb",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:d3b166fed222fbbcfe70e7a3872e8c1ac02a5ccd2cbf8585bedd29c005f6a16d",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:33651be7274ff2cd66c9e23e7eb20d5d5ca9649aed6777bb8b0ec03dfc8b0707",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:e5a7688faeb49b5ba135fb37251926c9d6c877c0793a1d88a786f0cd2efb03c1",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:91e79536fefd2f5f6d4dde0a46441fa610c410c721bdfbc3f804da5e2e8c77ee",
		"registry.redhat.io/rhacm2/klusterlet-operator-bundle@sha256:5524fe5d379b9fd4351243ccd3c0acffef58055e0782a527a90fdd711e8d5044",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:3e26ec915e2b9b21300d384594223ec3db3e885a627ee433df80ac9b04a5da75",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:307e78697fa8546af3c965feda55b3499a6db0c10612812ad849221b4d450534",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:039844b547e6e28f5e24b8d0caf9d26bc04ff61f44e3b3a5a6b4e6e5d4748293",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:ca5315dec677075a643cd009dfbe768eb6d1dea941e41e3b7ea88d1a430c2fae",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:5128091ce8551d630ba7afb4723d9433da3cb09f99eb06b1dbc4fee490faa612",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:096949be81c57a0f2d6834c6dbd3a39f8464d7687115da5824dd3acb58bf4bcd",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:1cd80adae36f3ceae5b8f3703d5761e82c1a5ed364b5f771f537dd6090d1c5a7",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:546e4497d96fd0ef834acbe7dae2c7383d2e5325b14cdaec2e631560d102720f",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:9811228cf63e2d85529410c0fcaa36999a25dcede649dc4d7d809b98b0a1c332",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:90a1f9a2db2b0d1ac78586e79dd0ea3e1ca0a20300e4da95118d0bb77271b39b",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:76402738a5a52397b164aa85850df9d5ffa446a2be4b0956d30120c1eade0848",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:4ed94efd3ddf14dc111a473c891469f7251cd94ccf2046c96524d17ca82c6602",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:1d08941f581ce9c39fde20ca1d6e17b2732b78504cd490ccaddc75e5db144a08",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:e99ff69879859d0bf689e052839c47d95e47da44926f86240d8f833858af77a2",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:6639ea7b8cf2d0f4a1df71116525c3383c9a55962403fee4bae461e1095d3718",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:37bbf59bce47d6d469b908003f8545b4e2c87b52b274da59daa1993a22fe5591",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:23dee295d93e7e917cefdbf0a6933bcf6136d0f44d723f03f0e533990ca8f14e",
		"registry.redhat.io/container-native-virtualization/hco-bundle-registry@sha256:4a64b511e6d455bc5bcbd821456b1f089cec9f902d90289dd89b8a923b09d803",
		"registry.redhat.io/openshift-pipelines-tech-preview/pipelines-rhel8-operator-metadata@sha256:0f4b4082d0b087da0a56cf43ce17763925ac2bca743bf1ef164a38375ae3c5a4",
		"registry.redhat.io/integration-tech-preview/camel-k-rhel8-operator-metadata@sha256:7290c901a716ce5285659c69cc212f678f5e5c203fbdfef062307eb9aa8eb8ce",
		"registry.redhat.io/integration-tech-preview/camel-k-rhel8-operator-metadata@sha256:684bd0865612137b06177054af618ac1cb800ec9e9f4fa08824068e71e5deb37",
		"registry.redhat.io/integration-tech-preview/camel-k-rhel8-operator-metadata@sha256:bd035c4c61c5dd8a7e8b9420a296733c5c410aea5bf993f973d61dcb2042e998",
		"registry.redhat.io/integration/camel-k-rhel8-operator-bundle@sha256:d133ea9aa66646d493443dcabccebcbdb1e7a70dd84939ed772e912f4b3090b5",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:b1f90409e146f456c35927db9fa03681a06bc6588993820b5e7f23f41bcdd13d",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:647cf102d13ef1e4db397f3f4a661ba18874ced4fe8a898872826d4cefe425e6",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:864547fac0966a74c2e1642bf7f4718fb8e449a094511df0dc0ae9a7119caf29",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:b494c1eb911e7949218ae8851b3df4d6d1c82c30249b032a16410e95b22b1425",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:c7ae914f498df12fae7e2b4ceed55ccea5047af7e0328ca480c8ec898d2e0d90",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:781665631ab00e0c1a68b932624d84da76b65e3f140cb6561084ce7b5d0b68ca",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:4084782075db365dc60e1cd142e60d699f2a5c28d7ed4c414b3fef5e79529cac",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:f51bf79d93c571064a8038cd7ab40ced67ab87d5e0063afe3dda47935cb1eda6",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:bb0f2c2f421501e12d559ae7d9247d084d020ec5a78c1b42f065edbccf45d69e",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:0ac3c8490a5426c9e0ffe5d48823108b122420b6991c08d342ccab1ac7559f21",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:262beb6c4826ce26a43cafd2962ac058c6053606afeb64132e031188e6d7f929",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:c55434f36e23c9e1edcc6e5dbc843c380745227cb7a0dcea38c34e94e616640a",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:c1cb56d6188bf6622bdb05af087092ed9dbe3cc92645cba3e6289bb7d7eab7a1",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:dcc60de51a12452dd1e470710daf2312917b720c2f5ced8874d4595d8760afb0",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:7b43ec0c862eae649975e119bb122cc631be80ac117631dd4d3a31c761aa5810",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:e30456268966cc10cf3648c386f680f728541cfd522123e59ed276e8e4bed55c",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:3747ebff5550e9c04abb0f9f1909e9df9c0dd7b85a791994248a29a7f399e199",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator-bundle-backfill@sha256:70bb9a7a04774ff386c44889ba94811ac49829822aadd0970b1a55178afdb60f",
		"registry.redhat.io/openshift-serverless-1/serverless-rhel8-operator@sha256:7ff51fbfc38689b72ad6c8591a8c7d21b32f48da3a6ca0ed27c3dc08af1a1738",
		"registry.redhat.io/openshift-serverless-1/serverless-operator-bundle@sha256:f950157a8497edec98f412904ea80359f5b081873b5a79e72289d8db175e6da0",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:49b50e2b4cfe341a30f6d4f8e8a6d411da7926335bd22a2397ba37acaf1ce756",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:39b0ba26ea73c6ba54c8ce0c86c2563a3284e5690f091a0536c9722e4f11d62a",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:31c7dd275bc4c2b0d9ead9c2002963485791b279a90174191826236ac624a44b",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:720fd4d865f4e4a404d9a7846d5608b87d540f87a49cb5161ce383be0233d4b4",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:a4f5b333f7009001bc27849f752aaf71daf03a534e65b59b88463e2b8ddce8f4",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:6269cb4e07a545c4b06557e7a3f3e6969c51cbc4fb13efcd593b4c045207bd06",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:a21153027b7a1146752cabb02f53bd2e7f5b72af3af38bd822e214e83e046f31",
		"registry.redhat.io/integration/service-registry-rhel8-operator-metadata@sha256:23a0dbbfc55ab4bd141412f0e0314fc2f11b828ab0fd533378b087282f538176",
	}

	jsonFinalResult := "bundles_registry_proxy.engineering.redhat.com_rh_osbs_iib@sha256_5c0c280be1aa65cf5649894e26b64840cdadaf71762b287b2c0f7c6d5ee6d4a4_2021-09-07.json"

	file := filepath.Join(jsonFinalResult)

	apiDashReportFinalResult, err := getAPIDashForImage(file)
	if err != nil {
		log.Fatal(err)
	}

	// Get all and check if has any bundle that was configured to be deprecated
	// that was not.
	var notDeprecated []bundles.Column
	for _, v := range apiDashReportFinalResult.Migrated {
		for _, i := range v.AllBundles {
			for _, setToDeprecate := range deprecated {
				if i.BundleImagePath == setToDeprecate && !i.IsDeprecated {
					notDeprecated = append(notDeprecated, i)
					break
				}
			}
		}
	}
	for _, v := range apiDashReportFinalResult.NotMigrated {
		for _, i := range v.AllBundles {
			for _, setToDeprecate := range deprecated {
				if i.BundleImagePath == setToDeprecate && !i.IsDeprecated {
					notDeprecated = append(notDeprecated, i)
					break
				}
			}
		}
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	reportPath := filepath.Join(currentPath)

	// Creates the compare.json with all packages and bundles data that were
	// in the origin and are no longer in the result
	fp := filepath.Join(reportPath, "compare.json")
	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	jsonResult, err := json.MarshalIndent(notDeprecated, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = hack.ReplaceInFile(fp, "", string(jsonResult))
	if err != nil {
		log.Fatal(err)
	}
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
