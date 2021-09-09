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
		"registry.connect.redhat.com/noiro/aci-operator-bundle@sha256:281e5b5458b4f96e23d828da2f3320785103183674127549cd9fefdfc04ba304",
		"registry.connect.redhat.com/lightbend/akka-cluster-operator-certified-bundle@sha256:e7dc6a53e75a9416f0619998cb1ab12aef98f2e4cbd8f10bc4f60bfefa553e82",
		"registry.connect.redhat.com/wavefronthq/ako-operator-bundle@sha256:6a41ad6e74de7e5c94af7781a001c4501fae4b7b5505dcbfd75b04053af1e1ce",
		"registry.connect.redhat.com/alcide/kaudit-operator-bundle@sha256:8b95b3769017ae44a5ba4bfaac8bccc07d11f2afcff0ce03af46e71cb440f9f2",
		"registry.connect.redhat.com/anaconda/anaconda-team-edition-bundle@sha256:43327356ec622d12126d5d13ae9c968dd7cc9e8555eadbb8e212d0b37cfb25a3",
		"registry.connect.redhat.com/anchore/engine-operator-bundle@sha256:aac8eefb589ceafced25b345b5f4d5ae6dbb7c2952ffb1985227ebee9ace1714",
		"registry.connect.redhat.com/appdynamics/cluster-agent-operator-bundle@sha256:cebac3376fe72321aefd39f816331f37242f602e1133ec2f6d825cc128a8d066",
		"registry.connect.redhat.com/appdynamics/cluster-agent-operator-bundle@sha256:ef6869ae9ae4d83e794b4e83e591609aac0c1efbdf27afe98eac1409658c532d",
		"registry.connect.redhat.com/appranix/apx-operator-bundle@sha256:126965743834b9d3fa15f354f41fe47618cea51c6f2ab88d3401b62b0362b1b6",
		"registry.connect.redhat.com/appranix/apx-operator-bundle@sha256:0722fe603686fa004bc0047f4535435ab9005ef4e9fd96820d330c1c17908259",
		"registry.connect.redhat.com/ibm/appsody-operator-certified-bundle@sha256:6eae693fe4d14a6f72ff49e151aa5fa9bd0ebd75fab08effff7c55aa3c1ad55c",
		"registry.connect.redhat.com/aquasec/aquasec-operator-bundle@sha256:7efc06dd8e946188593079d6100187034f39298bce2c13ab50cb1e0747a9e608",
		"registry.connect.redhat.com/aquasec/aquasec-operator-bundle@sha256:e3b33c305cd178151b2d61fe3fd7e17285179087e65fc7d9ff710b37fcecc256",
		"registry.connect.redhat.com/aquasec/aquasec-operator-bundle@sha256:c367a6e2000ff5eb18363b3008930e86623a1ef74d0236cdb9423b63468fdfc2",
		"registry.connect.redhat.com/armory/armory-operator-bundle@sha256:f652675af906bff73b5658d588070328aa9c3fd1f460c35c72e12de504a3050e",
		"registry.connect.redhat.com/openlegacy-corp/as400-nocode@sha256:e1b06b21720e5ebaa68fb00eb8fe7e44d1574f73164528ce8d6c560ccf9beda1",
		"registry.connect.redhat.com/atomicorp/atomicorp-aeo-hub-bundle@sha256:5eda67f2842f0ceb90979716597e1c3d94eddcad67e1014bbe56f7e8474d8d71",
		"registry.connect.redhat.com/triggermesh/aws-event-sources-operator-bundle@sha256:efaeab6112c4ca88ba5427936eda7e3b316e22060fa6ae8b920410cd9a3a6c4d",
		"registry.connect.redhat.com/triggermesh/aws-event-sources-operator-bundle@sha256:b41abde7767e00d6ad016739df7e6c07fa9ac21e096c9fd6b42bf1aa08abb162",
		"registry.connect.redhat.com/triggermesh/aws-event-sources-operator-bundle@sha256:b9d8f05aa217db6847aed2b06cff677777bbdd5606191b832ff53000dea489cd",
		"registry.connect.redhat.com/triggermesh/aws-event-sources-operator-bundle@sha256:76d525c7a995ea48ba06e9f1aa332603869aff8be72f2446b53bad92753fcff4",
		"registry.connect.redhat.com/bacula-enterprise/bacula-enterprise-openshift-plugin-operator@sha256:fc9f5368c10878f1e7fc0e4968c2186d079bb233cb769b7375202bd36ccd5610",
		"registry.connect.redhat.com/ibm-edge/behavior-analytics-services-operator-bundle@sha256:79520ef98e9cd360fb2a01519631e3d3c488ada6d3cb8c68334c4f517750c0ac",
		"registry.connect.redhat.com/ibm-edge/behavior-analytics-services-operator-bundle@sha256:c73665a97beb2f5def36868ecd42cf870d7a92a0f2f728099ec05dd3abf584d8",
		"registry.connect.redhat.com/blackducksoftware/blackduck-connector-operator-bundle@sha256:17b255728650f2fdf7e7f289a9b8cfdb96bc2d283709637c0b540690512dfe70",
		"registry.connect.redhat.com/can-avanseus/can@sha256:d957c2bb195782cd0c5a34effa2e4a01372b35d559e3877d938e8acb31f5ff85",
		"registry.connect.redhat.com/jetstack/cert-manager-operator-bundle@sha256:e0898bd5df18e2dccbe2d648cc4533e064e584166675e02260cb856817ece844",
		"registry.connect.redhat.com/jetstack/cert-manager-operator-bundle@sha256:a7cb0f2a1b263c432727d1ffe7ec8d0be470fcca26e1b568eb9d0dc64e96373e",
		"registry.connect.redhat.com/citrix/citrix-k8s-ingress-bundle@sha256:3b9f5cddc5c4c048f1be9339b50ce88bd042ecacc3259811e61e9fb0e45cafbc",
		"registry.connect.redhat.com/citrix/citrix-api-gateway-bundle@sha256:56abc00d5ad2221471924f4bec169ac65d92a61f1ac87dff4d6dfddd10496fd8",
		"registry.connect.redhat.com/citrix/istioingressgateway-bundle@sha256:16a717fc3e153fcfb42d9a47da732803f21341cbec2ae3bd1950f33b6043619e",
		"registry.connect.redhat.com/citrix/istiosidecarcitrix-bundle@sha256:98f70609bb8a1519876b9c103a200abc88b74dfc0ef9c8a0c0aa464c1e8c463e",
		"registry.connect.redhat.com/citrix/citrix-k8s-cpx-ingress-bundle@sha256:58f15e99628043753fefa5c8df190cc90e82a6bd3b00aed3c52f0fbf69d81039",
		"registry.connect.redhat.com/citrix/citrix-k8s-ingress-bundle@sha256:1cf84c905b444c613f67f5b2a7b80015226b6a99c5723503e87c795a719e6347",
		"registry.connect.redhat.com/enterprisedb/cloud-native-postgresql@sha256:4ac5d2ae655403f7cb1cede3d4bab2adf1d996b839bc289b76c5acd3299f3552",
		"registry.connect.redhat.com/enterprisedb/cloud-native-postgresql@sha256:01f8ef65ccb4fd5bfe699fcdc62de4cffed5f3d35dbd0eba073b353ef1635fe6",
		"registry.connect.redhat.com/cnvrg-core/cnvrgio-operator-bundle@sha256:56f6c78611a26e4ed42e8228067ae4851799224f80d965bb034d7fe36f32b65f",
		"registry.connect.redhat.com/coralogix/coralogix-operator-bundle@sha256:ae2b55af16cbd21e45ab4bd4483c106c03d77ffce2be61f92f59b36a98c82f41",
		"registry.connect.redhat.com/c12e/cortex-certifai-operator-bundle@sha256:7beaacb9261b0ab823443b4f7c3f3c3bff8250caee7f1ca68887e2457e8f963b",
		"registry.connect.redhat.com/c12e/cortex-certifai-operator-bundle@sha256:2ff2df4e61105cb9cc1905f504ed0e74907fcd43f85d200f729e464e824e928f",
		"registry.connect.redhat.com/c12e/cortex-certifai-operator-bundle@sha256:26199886640c352804df4a634f5ea3e45c3c4d4e41f241e9ef36cd4b3536be57",
		"registry.connect.redhat.com/c12e/cortex-certifai-operator-bundle@sha256:c85d3cac53bab7690ee87f6a8fc15e3471e6ceff7feecf9749fad89a1e6a12e3",
		"registry.connect.redhat.com/c12e/cortex-certifai-operator-bundle@sha256:d69cb1d66cba931eb04310b8917a41cf427ba70ad5f422478d812b303391a8c0",
		"registry.connect.redhat.com/c12e/cortex-fabric-operator-bundle@sha256:de8cf8423ab8ea22f2090947fb51c3091407fcca623d5a83137b266c43e00f6d",
		"registry.connect.redhat.com/c12e/cortex-fabric-operator-bundle@sha256:90a89c97f73301c7f82424ca069677217faa61095b41401bbd6024b72dde108c",
		"registry.connect.redhat.com/c12e/cortex-healthcare-hub-operator-bundle@sha256:27cb3851d88ee293c260e452670cad361800cc50b1c9ff98cd3617113b13d117",
		"registry.connect.redhat.com/c12e/cortex-healthcare-hub-operator-bundle@sha256:40fa05e3729d61d05527d36e307b3d2b2299defae8fac6a08f11d676c0b5dc2f",
		"registry.connect.redhat.com/c12e/cortex-hub-operator-bundle@sha256:53ed857edafa5f90ad45ee3c67be454e8b42cc8a12fabdfcb87b93fcbb4bd1fc",
		"registry.connect.redhat.com/c12e/cortex-hub-operator-bundle@sha256:8330ca92f9b71fc4380a01206185838b3a706d00d156726567436baac65291a6",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:cf4a0448fa22963d711bb2d3af8b56e3c356073b76233f33421121403893a859",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:a2c9323d78e4a2e83810eb9df6e2f364356cd5432cee9bbd98c94784fbc27468",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:9462f3735254b047f4b80b539797b83c31faa0ad923fe4a351dba693e51cadef",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:1ec6fc8724eed1b9260eb3765a6280c0f1781bf6d51e998903cb65f1e20b5d73",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:10ec7cafa6c4be66ea634b88e10a2b136fcd3322cad00309a7ccef455679c6ba",
		"registry.connect.redhat.com/couchbase/operator-bundle@sha256:78656ed0df41696e67429609e3262ec31f5ca9f5ca946300e6a4a6503c1bbd18",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:bac2f2b7378030d82b3385fc3f441430a179a611aadac8a65d27e304114b5e16",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:93ac2ae304e90e37554a97fe7705dd3b9cdb9ab89c29961a0094601908e2555d",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:351a5012eaf8a3649bf81f1c3a737cb5ae4457e9df8078a7229322839db567e4",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:a065ec9f129f79e21881ee4644c1e34c3f13b90bdc76d72cb76bf8216d45ee44",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:730800366a7127b70c07c8298d9ecbdbf32c248ad09198388fe1a0f03b3ee4cf",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:ac3d5a0fc91bf7a6b2b06f6a19e42c219bb304681dc3f50c3aaa3d9a206ab1c8",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:14aa8a718a050baa0fef250d3385d18f408c572f38ee53528ce4293f4db30757",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:0bf222bc4a737f1c9c20a8d6129177cf3dd2bd7d52bb08834518a6e81ecdf0af",
		"registry.connect.redhat.com/ibm/couchdb-operator-certified-bundle@sha256:b660525be732487acdd72219f301c9d572dd29d5d9b1e1e771da90b1d3652a21",
		"registry.connect.redhat.com/citrix/citrix-k8s-cpx-ingress-bundle@sha256:88485817f37b79ef5d8cf84552a956d7989504c2f031f21ca6f487387074ed27",
		"registry.connect.redhat.com/citrix/citrix-k8s-cpx-ingress-bundle@sha256:5c437ab48671036215dd807ad5c92e07df59b90577a0e48e68d6d193bcd23fc6",
		"registry.connect.redhat.com/cyber-armor/ca-operator-bundle@sha256:c9d7743181b4811203eaf7f977949ceaa369cfb2407d878bf8677f3d0536419f",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:b6efb516d646752d2309c3791a1d094d6430f9230ebfe9694442ff2eea7d5aa6",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:2f9440148f9741bc7ebf6d59d02b42ab4e8969ead1eac58251578e2ef1524400",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:61317c2c85660fe4291b3945c928df375890ad00b3168b72bd71b787ef4077dc",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:cdd24dd021bb730168eb11b796872975d84fdb471456b745d03a30392dc38d2d",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:403219f0791689e2f3484f33be08e243dc2bd9e22ca962c430ddd7eee2caf0ee",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:af28ac6d3c7b2f0267b54554b777a6475c6e61fb32dc09d0e5018a0709d363c7",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:efff12d1b48af0f69de56abd18edc717c6a647f52c7e34b41bc38191cffcc363",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:252eb0f3b3cf70b199f209a8ec03ff0d1c99c4cc0a60af18687238944b6e5140",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:bf21e966e38761ab59e74271d1ca154f75e2a1c85d6f4eab418fb6d5261c9d1c",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:20211956d089d8958f08e103304eae586144cddddb5cf17f71fc749cbe6a4da7",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:16f043d2f31f879b1aa6cf2385ca873b45850f238c32370b661f2c8ebc7668be",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:29de0c8f38adb379957ec2564eea036e4f839980377e52c49f8b462e3033c7aa",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:ced5bc02c7bec13ba5d10b5fcfc559af8499eac9bad9688401d1caf8beb6d3b2",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:2098ab48bafee24a19a6bd6592a2b177b5d863e9c1d2519bf095325dcd4476e1",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:9afce5e1bbde0725580c579ade487aeb2373f647ab7f40cc4d0bd061f3f887d6",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:e66be49a23d2b01b81e76c49e7069ac549cb6d44293b94f8b5a2bf76a54054bd",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:10307151bc8be054b677438353bbe702ce57cc6ebe6954ebd8e00eca0356c852",
		"registry.connect.redhat.com/ibm-edge/data-explorer-operator-bundle@sha256:5fac1e0a1fc410dc78572fc250bc3d15c121c2ef7aba8496eaf87ab20b4de91a",
		"registry.connect.redhat.com/datadog/operator-bundle@sha256:c4fc64e5ed02c594b876983564ee96123ba255c2376caf9d9f5584ed0a273ee1",
		"registry.connect.redhat.com/datadog/operator-bundle@sha256:f0b28286ec3daa7f978c6490f3d61b3fd3db84caf2037eb9f8bd4b46ca1b56cd",
		"registry.connect.redhat.com/openlegacy-corp/db2-zos-db-nocode@sha256:f801fcb8a5c445ffce1dadcdca2f557f58ab40538721646320cff9401523561a",
		"registry.connect.redhat.com/dell-emc/dell-csiop-bundle-110@sha256:36a685117aa33ae94e6e768a10e24f6a65b2cc6f14cdd778948108e613ae5abc",
		"registry.connect.redhat.com/dell-emc/dell-csiop-bundle-110@sha256:0fe8dd27ddc36df73517195b9d4983a4c54e3036683c83952a9514f335ac557d",
		"registry.connect.redhat.com/densify/densify-operator-bundle@sha256:b369e48b59bcbd9710be712f601dfe00b09e2be24047f32c91b025a9336c88b0",
		"registry.connect.redhat.com/densify/densify-operator-bundle@sha256:6c37c211385ffad721637ea1a6bcda46e4aa9aa46efdbeec6f75e9b9bd7ff75d",
		"registry.connect.redhat.com/densify/densify-operator-bundle@sha256:b3f96cea8295b88e41a54f80161ac32e79b28f5b0218f93f1f042ff1b9af847f",
		"registry.connect.redhat.com/labsai/eddi-operator-bundle@sha256:c848665b6cddace5bb008c642a90df95ef75febe83fd01d3aa66c6d3eb78ed90",
		"registry.connect.redhat.com/elastic/eck@sha256:9587db325f9cc13519f16e9fd675e8d74c80603ea6173ed0c25f74e72a2744a2",
		"registry.connect.redhat.com/elastic/eck@sha256:c5cfca9f272137c630239676960afa07b64edab4ac2a7be5dc6cf27e17424bfd",
		"registry.connect.redhat.com/elastic/eck@sha256:bb54021d424a684456facbdf599326a6a4fe3c4cde60ba478355177e215ff099",
		"registry.connect.redhat.com/elastic/eck@sha256:6251f79b1f7b589ac83764454a153b885771a481bd55d7d1c17bce6732209d4d",
		"registry.connect.redhat.com/elastic/eck@sha256:30333a951cdac81f368f047aad9d8b6b7261bd1cc8a8f7434c69d2ff103ecb1d",
		"registry.connect.redhat.com/f5networks/k8s-bigip-ctlr-operator-bundle@sha256:5a11aee22850cd5c1bf3c627ad4cb58af3f9d3115dbd73a1074814b3eea2a5dc",
		"registry.connect.redhat.com/sysdig/falco-operator-bundle@sha256:ed6d3b83f8ef2c7b2f97ab24a442f211d225f93d516ff6acba55d0566ec3cd2e",
		"registry.connect.redhat.com/prophetstor/federatorai-operator-bundle@sha256:b61dfd3b015cba83daf5dadc1e502a315eac57e1e9538c818099248138f70abe",
		"registry.connect.redhat.com/fujitsu-postgres/fujitsu-enterprise-postgres-bundle@sha256:adb3e5776fe4376c9a6c0555f3306482f4cecd3b6fa6cf708252309d877477d3",
		"registry.connect.redhat.com/fujitsu-postgres/fujitsu-enterprise-postgres-bundle@sha256:9a4e357640cff1fe43415b2d587dd3718016734950f735e9454650392be0626d",
		"registry.connect.redhat.com/fujitsu-postgres/fujitsu-enterprise-postgres-bundle@sha256:1c5f3b645ceb05266cf8cc8e8a6f88e8400a855f23277f9b79e8b9f2ba278094",
		"registry.connect.redhat.com/findability-sciences/fp-predict-plus-operator-bundle@sha256:e4bb2a292033e4eb6de2eee46395465a20f8b067b59fde597eed009be5e93161",
		"registry.connect.redhat.com/gitlab/gitlab-runner-operator-bundle@sha256:1d5c3626119ea8fd92d291ce199515e6cde463d5fbeba831b226966b72e931f0",
		"registry.connect.redhat.com/gitlab/gitlab-runner-operator-bundle@sha256:9a88454a1cb9fe76e6a3a788c0d7a9015af3cc1fd42caf63ad46602959fba2b8",
		"registry.connect.redhat.com/gitlab/gitlab-runner-operator-bundle@sha256:abe1504fedbf38bc1cb45e5393216bae9cca95066ca84fa730c7e8b129930074",
		"registry.connect.redhat.com/gitlab/gitlab-runner-operator-bundle@sha256:e2a6efc154251480e17eefb2b3159bc125d3258fe84f0f981c19f36c29740adc",
		"registry.connect.redhat.com/nvidia/gpu-operator-bundle@sha256:3a70920f1aca227ebe5b1db44125c4370bad00379a941f0bac301d61ee112ba2",
		"registry.connect.redhat.com/nvidia/gpu-operator-bundle@sha256:243b8f0bbc2bda6bb13395f165fe5ec01999331e9b61878686d2073df3bf906c",
		"registry.connect.redhat.com/nvidia/gpu-operator-bundle@sha256:0e69c29d33d5ab9d6676c6a6d5b89e57caddf44090bbabc74a12fea3005704bd",
		"registry.connect.redhat.com/nvidia/gpu-operator-bundle@sha256:5ec5a5cbaf6e667e761d3a6273c44f5facf29dabe2ffe502f08fe345536c8dc8",
		"registry.connect.redhat.com/nvidia/gpu-operator-bundle@sha256:3a944050b7ba261b5451e639165eb230f1afbc3be7624a7d6039f138b09cb19f",
		"registry.connect.redhat.com/ibm-edge/growth-stack-operator-bundle@sha256:04327840791db3cd6e2ffa48084a515dc763f2119ca678f9d63cf7a11b2f4c82",
		"registry.connect.redhat.com/ibm-edge/growth-stack-operator-bundle@sha256:328b7978ab51e7019e118ec1777278e5d5c2f74040a292f650f7bb8b3f1917cf",
		"registry.connect.redhat.com/h2oai/h2o-operator-bundle@sha256:a31c191ab992b727ade63d560c284fe953ad8363a5387a88e3989c29b21e4b61",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:e71b04d8d60de221e857cbef51988caf6f11e938f65b10254eaaf96696ff306d",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:8f3a36528fa5bb601e62d071e93c4026c734d840e43da3645bdb2cc1a5ec75af",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:8fa5c8e02b13ca0fdb6a36aec61ff5cae03ecb8a97a6706896b894767adb7cae",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:aa8d5f74a9bb2318aabfb9cb71ed01f2c88ae892b55169872716d05e265b484c",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:cc9f6bb951d46d2ff95c8ae8780d732752a683a33c51795684ade244f6d5b84b",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:094e6fe764454f6623284e79403d98678abc19b1f4672dc079bc3a563213b1ac",
		"registry.connect.redhat.com/hazelcast/hazelcast-enterprise-operator-bundle@sha256:4090c5206fa7c0602ff0ce4cca18c632b97d294a70ad8b0d2b68ef1081927ea6",
		"registry.connect.redhat.com/hazelcast/hazelcast-jet-enterprise-operator-bundle@sha256:ec7ad7ef1acfb3f2c38aca831742e8008ae41d5ca1fbea38bbd3714eaafaca86",
		"registry.connect.redhat.com/hazelcast/hazelcast-jet-enterprise-operator-bundle@sha256:5879624f21f1d7b263d280bad0f2d209c5fbcac0f624c42b102036eb9f5da33c",
		"registry.connect.redhat.com/commvault-hedvig/hedvig-operator@sha256:a68668ac612e5240c61c22f05f7c5c53054058d2caa624c6a49576e61de5d193",
		"registry.connect.redhat.com/heremaps/here-service-operator-bundle@sha256:5a48a5194566573c5010a2e3ebc79f834652a777253856b2bc1e2e6c5c0f26e3",
		"registry.connect.redhat.com/hitachi/hspc-operator-bundle@sha256:ea73bbd83541c454567acf9dbfc348fb7d72d97eaeeb584f2a452cf379621189",
		"registry.connect.redhat.com/ibm/ibm-block-csi-operator-bundle@sha256:aa2f7b4b4b2d489cd9671b0b45d696ca2fcb8f76ad2b2551a06475917b9fdc99",
		"registry.connect.redhat.com/ibm/ibm-block-csi-operator-bundle@sha256:d78c62ef65db4858374cb7bc8434ef6442f6aef306475eb482d130abce8db971",
		"registry.connect.redhat.com/ibm/ibm-block-csi-operator-bundle@sha256:4e6b7b6006f6ad537b75fb681805003d4873230ef2706cab3333c4567502361c",
		"registry.connect.redhat.com/ibm/ibm-spectrum-scale-csi-operator-bundle@sha256:70f310cb36f6f58221377ac77bfacce3ea80d11811b407131d9723507eaada42",
		"registry.connect.redhat.com/ibm/ibm-spectrum-scale-csi-operator-bundle@sha256:26e507a96b0473964ab18c84ec899a51c05301e956c3968dae8a0243842f9b5f",
		"registry.connect.redhat.com/ibm/ibm-spectrum-scale-csi-operator-bundle@sha256:042e43455cf5b9fe1acf408661bf8163a3f561a22baa3ef82e2580f447396a81",
		"registry.connect.redhat.com/ibm/spectrum-symphony-operator-bundle@sha256:6920ea67b4a0387453b95376e2d8f6075eb07b39b42ea3fc7f3d256b68042ec0",
		"registry.connect.redhat.com/ibm/ibm-tas-bundle@sha256:5bdc5d47591f604d221e74f71efd091873df76198e64cf3ed50a85cacc11c9dd",
		"registry.connect.redhat.com/infinidat/infinibox-operator-certified-bundle@sha256:5a8ca8be977bf98a07ac9ca02a2e4132d88e1f2e71130cbd5ff1acec231e4c32",
		"registry.connect.redhat.com/ibm/iao-bundle@sha256:a05da58a27648ee35f38a912cffedfd8968146fbb3eeeb618fe35355c08a8d8a",
		"registry.connect.redhat.com/gigaspaces/insightedge-operator-bundle@sha256:f4cd0ec77bc5f19409242c4e5990447281b6bd60b8d0eff848744a431c73cd22",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:e3bbdf42b6d8e4d5d41793209ebc8c883efee6be03273bc3fef3ce3fe0dc171a",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:e827ba4c955ecf3e52d72b2ea012a1340daf3f0d7a1f1923e2ba46d9e291b71d",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:430f28f86bbc3e14e226133048c0fa27b5a446087329d5ec89262d1db5450250",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:30cc380829d5185e535c6ce2862fdfbe725694e638edade24d1d2ead76f063ca",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:bd5065d9525fb3a5c9520200e074d7d95781b19f41f0bce6118c7c0367f4e015",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:a01cf756a40c09923a1f00a7516d83a46087731265dcc5ed10a7bbcd8bebd8e8",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:89cc136ad0345205fdb4fd32a6d6c73a02068f6a5a9e0669a57c4da5d9624285",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:c74519be65015d28508ca3b260449a0365f97968f17e20b48c7b338247c835c3",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:62e44a3a7ad63ce183162ad49de72ddd587358251b91a890cfebdb72945c140d",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:0e445f78b7f921890bfc6f013a0ed5b9fa0f9b2cddf9768e3a8e83d52716b043",
		"registry.connect.redhat.com/instana/instana-agent-operator-bundle@sha256:e007bfb95d0e0498967a466dd482002181f4ae797a98d2b5d9bed751defc04b6",
		"registry.connect.redhat.com/ionir/ionir-operator-bundle@sha256:41d3d00ce48ca90dd3b86e07c2da216339d2b4e1dfe6fbd90708aec7d3df6697",
		"registry.connect.redhat.com/ionir/ionir-operator-bundle@sha256:7642cc52a32fb09e7467f3a1fb7d7c25fdbed423faf47e3bdc962c1916f69909",
		"registry.connect.redhat.com/gtsoftware/ivory-service-architect-operator-bundle@sha256:0587893f027c7b58ea169f9f2587a561ac778db502b0e26a7bf3377fda83b85e",
		"registry.connect.redhat.com/joget/joget-dx-operator-bundle@sha256:86591e38d2f64c6a9557ac5cff6383e54cd593002bbfd6508bb5337ec310596d",
		"registry.connect.redhat.com/joget/joget-dx-operator-bundle@sha256:484577c27bda23181ca94516e63076bca1fcf883f041eef996d5ba61ef39ccfe",
		"registry.connect.redhat.com/joget/joget-dx-operator-bundle@sha256:2f28ff53962f20a92565ee074b783ce62aac5162bf6b1e3ed4342c78edd087b7",
		"registry.connect.redhat.com/joget/joget-v6-operator-bundle@sha256:3a06e6251177f3585d43420ad647cd1e6731c959cf37b9294206b2b7ccbb67b6",
		"registry.connect.redhat.com/corentrepo/corent-operator-bundle@sha256:7f59b09438a61b79ef78872c6e7e169e4ef3164a6c5d04729fe45e0890e7bb36",
		"registry.connect.redhat.com/kasten/kasten-bundle@sha256:b75d6812f5cbdfc1a26a8d18b6877f2bba5bd239f03fccd246933f6859635102",
		"registry.connect.redhat.com/kasten/kasten-bundle@sha256:051fe316928519e1174758860da97389e0f2a13cf64efc8ff016a942e8e2732f",
		"registry.connect.redhat.com/kasten/kasten-bundle@sha256:a4d336e19f34d7e4c6077261cbe42d8bda2fa9044a7dc957e4d5c996dfc4eb13",
		"registry.connect.redhat.com/kasten/kasten-bundle@sha256:aca1212f1ffd5d2b76817b0f29a76fef27501cdf5c44798cb5363da9b6397051",
		"registry.connect.redhat.com/kasten/kasten-bundle@sha256:d2290057a7c9e3ad365bcc21d3a8c88c6917ae9bf3d2822b31c9372b705156e7",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:3d08fbc3203b68e953d71cf21d2311e2fc6e4529870274535cf1bab1e6bb609f",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:46bd2d9a110bb8964d675bec5863de8dd2e5dcdf7d6b97a0d64a8a543fbeebf7",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:4625d639ed8bc07b92b9231b67f67ed6c1866194cd194a02469e36ef7706da3f",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:d27785567c5e0db70d99ed7e220a5a58e6466384cd6103849ff0dea6bcfe5c57",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:51408a5ff0498bff23cf95222849e80311a10e3b6575325649e191059c4e8521",
		"registry.connect.redhat.com/trilio/r-3445381-bundle@sha256:1371ae42f57e443d556eb573342cc1a37f00f3b707b6a7d76b3be19a05bb2bd9",
		"registry.connect.redhat.com/kong/kong-offline-operator1-bundle@sha256:7e04701bbd16aa9cab693a99b7e6935e720ef35f0d5d830defffb8fec50b42d4",
		"registry.connect.redhat.com/operatr-io/kpow-operator-bundle@sha256:b18d7d949dd02bea3becd844f863889d110eeec8f383b20d34d0928c2a140c55",
		"registry.connect.redhat.com/arangodb/kube-arangodb-bundle@sha256:f7cd48b52e0913ca13968d396ba7158cb94344ad762838574724ef5015153945",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:77737935963af3be8b1354959037e7b6948710deae1150f0aa724af4d240aed0",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:fce2e20216b3adcb191be3b55f9a5e4505e021a0ffd495a89ef00466f4ada722",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:338e1b266542af3ff3bc7f628477b1f9714313cc7b18872141fbab7bad7241a0",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:a488e8afd4b82f128edf45f36f1c75c9ca6beb37855bb1d5009fc6dd8c9f3434",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:28def796348a6830ef800f0dcdd7a669c01a34f54222828f24b00d419d945917",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:afe0430ae7f0c5df27e4a3ca69ea25793bfd03870915221510c3d194e4a987ca",
		"registry.connect.redhat.com/kubemq/kubemq-operator-bundle@sha256:18ccdb6da4534cf1e10c831e392df7252273ccfa763fe7251391a8bd0bca4571",
		"registry.connect.redhat.com/cloudark/kubeplus-operator-bundle@sha256:a1944c0e06d995361c72f03b8a975a646dd25935a0312c5209a03cb1a1732256",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:f8bc2060cc0e1ec4bbae7fed0ff0013c4ed936b85413db3011f701bdce2b6367",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:1c09b241712469ae34e4017e9481e3124cb6ce7037b61a44b62f2f7e0c10df46",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:dd364c7a85d6a97629074293a481534226353a1614b61fb2f1ff0972a4016dce",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:770c0b5a41cc3103582daeb44a0d897d578e03fdd8597a3a385f8ef78d5b6f38",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:dd05b33ddb1ff5fb9e31d24ab0f568974d6592d50e02908c7a2e9f0d680e27b9",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:a6ef9961d40403e2b8832df6cec9e27f35ae70ab610ed138cfd05e153888b498",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:505c038233894f5fd5203f1546544f7b2a0cca334b2e8cc055243f473cd22625",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:8dea6665a8cd0a40b0ad7daa52f22ff341ae29c5b592636bfac25e4767224324",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:64962f1aa0777ff7bec1605a7c13be82fa7596b3cc997ae62500b7714b75c084",
		"registry.connect.redhat.com/turbonomic/kubeturbo-operator-bundle@sha256:62250d515fa374b29f57381fea8f4f9a56ff6e00056b38476e629c91606c7369",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:1a400322ed8279e6ded73c7fd89c3cc5df2f74ebede9d192017268166a2bb97e",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:43d4b83cd44b220ff97974e49f5a451f93058bbf5ed52ef8a650f199bdc0ebdb",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:abaf5a3965617a0555757344b802e6112317de9c343cc018f15c74a6843c55fd",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:47b4b5c50c392e404ca01fcb53fb7ccaf56000ccaeeeded071662df29b5fb44f",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:12cf4fdab07a4a3d97489de54b80231af171494de7cd686d1b792a3643c15142",
		"registry.connect.redhat.com/memsql/operator-bundle@sha256:56849484480e43b9a599650ec9c6fba1459e701a3fac5c457ff9b3d2750838d9",
		"registry.connect.redhat.com/openlegacy-corp/mf-cics-tg-nocode@sha256:bf84c776f995025ffac87b50971ff401cf92badf987d5c34a19915fa5e124b87",
		"registry.connect.redhat.com/openlegacy-corp/mf-cics-ts-nocode@sha256:cc02375c897a1163e1adf315286d008488beb8eecdc0b1804867434c9448e809",
		"registry.connect.redhat.com/ibm/modelbuilder-bundle@sha256:cf8245accfa24c18ad9d27792b5bd2379fa476cc1f221c749e10f1b0f770f8ff",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:bd877b7c44e5d1d99596ecb061cf37ae47947a7411cbc4d6efe228a74a7b86fb",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:4474b295785b6a10e82cf34f266022dbcc70897b2c12f07024fb86f035994eb8",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:6002125e35b4e8b853c3825a136725515f47a569daaab60f6862bf12a0cbb469",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:e73abd7b4086129ecd1a5f48209af59124ba9d2699de0c90b393377fe9596e2e",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:bb7df5aa12ccca3dca2f90c668dd2714c8519c72e8f62ce471034deb5e589c26",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:b723279ffbfd735a83afd8db3424f8d8b2806906e483a6b64f9fe12f9e9615a5",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:b77778d5cf77fc1ab9a72814118af5dc43d44c7d0d802c16a7862c09370b0e13",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:8fa9cbb9d0454f11fa1023889513150cab1a743ee092d7f4110824e4512bf1a4",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:81f2e3748346dbe79ba169cedf855f48d8e5538db6f0d4f56177200aff23d36b",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:6eece28f85cc8722599979c7d460e88a8e433b9e3fee9b71285a826ef9e30862",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:ade7cee1cfce9d5a5badbcca507fdda3baa842c80f36458a2d6c3ecd5876e25d",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:5d01409483df2df74a1241d290349d612387b64aca09caef9e04161586adac57",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:e0a36fbe70d56aa4faf878b3a2d8f4196008d87ea9057f8509eaf4334b0eb1c5",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:7c3886d3658f55f572d71ac9c6b0349e040730fb9de2f357c11d8bf595aca8cf",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:92f5f489e8ad0cc7e7b10bd474a86810b352b5cc95db7347ce46a676cc4dbb7e",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:061187a5a8e57993f3c5a8cec379cc0552a3d690bbafb697d669ae6d192c3a77",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:04e407898806a44d17138656000aa7b6114369f23aafbb58768d73d3fc37e7bc",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:6cba2675c9bc425e0115c07edd1346b1935b117b0e2154cb2554aec87341ee4c",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:470019d6a6e3463df83904388995bb461fd7c6c2882d3bb7b0d3a531e053bcdb",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:3d2a384b4f38cffc9d14f7c9f53d12e55dc1e147116b608d263a7c04aed1e18d",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:e426e5dad74e713683d535e97a4e586cb898da891e9e61177eefb2db5742398d",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:b5504dbce2d6d8f4dfa7f67549ac094e8f84ecc807ec676dd3ab9544f9d22ef0",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:3e658321d10716e0163b5538c0f9390d309717618bc782c7d95f3bf58e92c8fd",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:0b16171987afd2205857a91ae15f34e316dac574faca5abfa13e0a4d025e37c8",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:9af7931f63d1280fcdbe9f33801b560db6b0b44860aa7bdf83e276e907d13a9f",
		"registry.connect.redhat.com/mongodb/enterprise-operator-bundle@sha256:caa0aaef0026a513d554a02863abb3bb9f49bdee32bc7fc3e749f03c12ebbacb",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:3ef303a68ac1ec0f5b6b8fcb4be8471180ee24e69ddb6e454ec00cac806fdfe6",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:f1fc06ee96cc7509e5dc94ef6cd87921fbbe4b4112d9561c14ee06a426c59f91",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:7e0a5c64a34ebad4b0b156b07e1b53b7fa6cc9ed9c456ff0a33523d6e4313e18",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:2c354e02f3ddfd9ef6ccac776b77603e60f907b10bac730e133845292500dd40",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:863e2cbbd481dfe2c70fdfcf866c161c250b95e5e9a320fe211a805c456c7e9e",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:0cc69f7bb40a40bf02485677bc01ae3768b3ae8015c4d3bd1c1d088ee487117a",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:c9a87ec0ec782af1bbd37851f1b15e639a36ff8e4b5a295b9886d7516fc7eba4",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:df5719e58365fbd71274838705de8e49fe52a30417960fef6e799e01debf6c48",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:a3150ff18aa21ddeb3a7e052b3bbc65616b62c1d557e910b13528c85c8b9196a",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:a8c0bb66bd911a2720df2cf08391b768e719c2e23344d58913e681066fbbb556",
		"registry.connect.redhat.com/neuvector/neuvector-operator-bundle@sha256:6f4ff463c67af42c1c3603e36e512a60dedb95775380a1a6b26c4cded86de5e1",
		"registry.connect.redhat.com/nginx/nginx-ingress-operator-bundle@sha256:5825614ced508b2d168fca5dc0d53e169c0c598bf9008dd3ee0d6bf75781d3f6",
		"registry.connect.redhat.com/nginx/nginx-ingress-operator-bundle@sha256:d53eea0345b2e18fb84d1208f5ca6c2e74c425403ba1611b5b1ee525315da0ae",
		"registry.connect.redhat.com/nginx/nginx-ingress-operator-bundle@sha256:3dc0dc8f785195dc30977d96d038b8101ddb79e4e7ea9375c94bdbb533d4969b",
		"registry.connect.redhat.com/nginx/nginx-ingress-operator-bundle@sha256:bb5088b3310edf490b5b72e89077326cb92ce6ad543330953c8fab92dfb6637d",
		"registry.connect.redhat.com/nginx/nginx-ingress-operator-bundle@sha256:39e36214aae291f6724a9e688b16d702bbe57dfcb06e81de6d0d5efb0a5864d6",
		"registry.connect.redhat.com/ibm-edge/node-red-operator-bundle@sha256:bf24a03362d30d0c31b19ae0a6f9cbb3340cbee7e133c5523efc32ccc963ce0a",
		"registry.connect.redhat.com/ibm-edge/node-red-operator-bundle@sha256:0cb06a7e47db1ff13ac7542f73041f96fb8e734438db5d4b37e17a69879c0199",
		"registry.connect.redhat.com/ibm-edge/node-red-operator-bundle@sha256:834370621a7d6f2e4ade42e5a4eb63625f5a8a4b02f01d23a7422761e07777da",
		"registry.connect.redhat.com/wavefronthq/nsx-ncp-operator-bundle@sha256:9c4d4a422db64d7248dc1a9823e19de2372d06670cddd6d10cefde966cf975f1",
		"registry.connect.redhat.com/excelero/nvmesh-operator-bundle@sha256:31d2c78ff1afd16b0e5db2cee34d8cb325c61bcf2f6c0f0fccff7899e97548d9",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:16ebccbf15157c2f7ba6d0873eead210f0c10fb4bce32e9f3fedcd1704fe7f30",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:8a3a9689f37c693a1a87bbe2c1f794df9d577fb048c3e4ef74d88e3d5328b594",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:62ca1e8c6114e2ebecd5664491c22c283f406d584507a3619bae3967de8c9726",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:4f49f111d6ac40466fb7f88be3a82bd8b52b8c9567d0a19cd4d21c4c51d06bed",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:1677f856c2bdf9947e00d4e58f52c52ff2c7b102d8fcfda7cc2184197aefe33b",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:96e43fdd8e19f0f339c518cd01cab0eadc2cf9a02fc4177d64cf437415d687d9",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:48d5a2acbdb1516738ee4eeff15a1085de92a8d0ded3f317317d64df8103ba6e",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:a148f6b5251f07de56e593477154200ac9a2993910c93e03455fc2587de2a844",
		"registry.connect.redhat.com/sonatype/nxiq-operator-bundle@sha256:40e3e2a969464b33e9f0873628a01e01e6c14959753f9006bf79a8e558e4ea39",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:f79bfc5a9a3544037537dd6119084b2c72644f9e01b739796c0cef578f99958c",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:3e6bec18d3fcb326ef63f81b8dff5f5d21db6fad489744c94542c75518ca638e",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:69cec4ebc1b4638bc289891c7493e71df23cabb301dcaf7ee503347a36bcd8de",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:ac144d0f1720655a10cb43727a3a05783ad08d9397c376c1dc1c9be12abf4764",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:3e51c8fb0633e5766a160adbd387e066216a43eb6d134c28be8943fa26df87c3",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:fafd2ae5bf0190bddea2f177ac1733f77f13607435e3e7b97c64e65f83aae71a",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:e0bd5cb48d9c8520a5ff4ce7c4e9d20feb32bc569df33d6c61206268a82b48eb",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:2468cdba4a093312d121be5389b40456fac45c92de4be8560da448bc225be8d7",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:ba710f12a4c97b1a1e851d2a47b1b009543bdbe5c2a769a613d2b25238c19cb8",
		"registry.connect.redhat.com/sonatype/nxrm-operator-bundle@sha256:003b936ea11ad47de6b6412a5229314860d82282ffe2138be67862dab987c35c",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:76c97a3d3feba26f2126c4dddb3d72648e0f4bda817880b4e13988c9fc69115e",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:e2ef787858ea4f716ea7112802b874b99028779f4b0636e6755039d5f9ed8357",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:8c4b0a6f8e465f5b969ea88d78cb3883a308b1c8c19f68e3c9cef2f6a0b40584",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:68dbfe15d1fd859b294bea3f88dbe94ff61c788b3aa923ad8e029ce52a35a2ec",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:bb0802549d55a58ab1edec52991de5fecbb0403f9f9544a5cc2da44cd995a1cc",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:57038e95b66d81649cb412066c70a20f94c35b6e95453baeac3dbc43b001f374",
		"registry.connect.redhat.com/dynatrace/dynatrace-oneagent-operator-bundle@sha256:fc750e6590bf2336144268bb09688a94e0baca2d5ebfb45b9f734b5cb5f68606",
		"registry.connect.redhat.com/ibm/open-liberty-operator-bundle@sha256:66987dd9d568c2b004a358c9fbbb5c89a74a7512bd43599b2463b08cf44724a4",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:5526e86dd60790139a6620e2464cb3450091dd0e6941600e9dab66a6fc79012e",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:f4a009d95308df12c8e754d05313d6674b3e508efb918e054e682f8270b761d6",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:6cb92a59c468246bb9a9060b7abce13ba9f2bb9b8d583a2b65837afdc291270e",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:ee798802b03e93ccc74762d5da175a0e6f00692ca7f2c94c16c2048862bee7ba",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:7e1160ad7bd60a804af78882a34f38a741339a8ea21375389604dac3985d0f14",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:6467f705bdc76f568941848ae6accc02d639b3a61a09d8555aed0b8333cefd1d",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:106f9243c74c6a1a2509988d284889beb5a079f4112bf0654fee635a9232fdb1",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:109ae78d718c0cc410d157ed4737d94445215a4e06894bdebf467bdc7a25d5dd",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:1b4675a3d6fe07e2d2b5bf88bdf130a2d3df8f5db5f0112883df9c06a2b9facf",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:c791ad2b333614b4409381e5898c1a48029104af25f722cc5c93945fbfe1a800",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:b29e83b16f2f1d536f3d369964dd7e892768a6a5b439915f85770f7c055bd6e0",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:fc274d39736785b098a52a375301aaf3b0f3dd4d7cf5a263982086d15996e816",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:9e558f919dcde8725e92e313fa50263273b2eb785fc4c6cbc671af63b2b0b7bc",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:a4aadd0ff8dbf7d22fa1be564d0d07c597b98b6525b1d3fbeeff07187612bb8b",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:ee74e62af554f6ee99d0e6c3e1e6d5550a99797d7f8cfa2a4c0a5f38652c8f40",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:55db08dd0f5020223d955b1befd9f8efebe2b5cc341be54e5d665f42b1d96170",
		"registry.connect.redhat.com/jfrog/artifactory-operator-bundle@sha256:6a998ac372b5b63e022db600ca799c275c90c12f7b630c0fa588d1169ce850ef",
		"registry.connect.redhat.com/jfrog/pipelines-operator-bundle@sha256:319b188f70a139c3f5e1a64533280a60388365a5585465d55c6a4e1db98fca28",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:c699fc5449544fec0551567347ee541dbea8344c00ec2ecfb44e7a67427bd473",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:ac71b64ccb38aa4724dc95a782c5b18775d599c54954193abf2ab58589a93afe",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:30d89756af5a4ce5ac9d1298b62a4984f87e329670ea70c5d03cb25a15809924",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:a55697133ce82a3626aa9e8ff06c0aeef1c57ae81768217513e926131177a33e",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:5fe48acf4230e2104ed876f362ff1b2ad8ad66f29e9fd41a1b855d611911ce62",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:c021eead94bf385c5f463a6b3e6aa76584dcabdf11d00eaf2ce5317429251ddb",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:3e07815f69fc3f36abda84131830e76c62eaf88d9b31ad4639c6b5cc89b1efba",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:38c5f9c1bb486e49d6829a3427d62e493e89df89a00f5d635f6c943823213eff",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:18f3ddc0970c0aef2f6b288369f9ed8ba48516b6ed01299972bd6820eb0ad244",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:360ee474b795d583155b19d514ee2c8a3ffdeb1f1a6ec883b94c37443559b7e3",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:32a498be4c7eee8f5d94806127271d93b3400e78a8928682c76b5ae3eded44ae",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:8a2ed2e02372265c2950c6fa6b5d929d2c2fed7ee2bae2424782d1abe19564a6",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:46e80952bb5267d09d96559ebfeb04f15f7c25b81481402e4ed68823826eddbc",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:211689acde185f0a3342e34a9f9df17a01bc1aed72795ff564d0db3ffb57e800",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:1646e0157cbcc443383e7869b7e276970ef61f36b089538193f37040919b2c23",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:c0585cad4530488abaed82c0391b6afee1620ea552a088f0631bd8c9bbeb73df",
		"registry.connect.redhat.com/jfrog/xray-operator-bundle@sha256:5f46760cc6a192e8437bd409f02f9b4f5aba2467c5a44e3fb215ce525d116d7e",
		"registry.connect.redhat.com/tremolosecurity/openunison-operator-bundle@sha256:bf50b128eefd6fb80e48f08aa8f6e6f3f384ca3d76641e7d2f5df73b055f6fbf",
		"registry.connect.redhat.com/tufin/orca-operator-bundle@sha256:95250ad63b44b9c3c4cca17514d4fa0396c7a8f1d0c5394383c70f4cee653cfa",
		"registry.connect.redhat.com/tufin/orca-operator-bundle@sha256:ec53a8fd3ac62698b27d52ee4e1af95eec2620457990de26b1673dde5ab75377",
		"registry.connect.redhat.com/tufin/orca-operator-bundle@sha256:3ea67ab24efcb9b693d57ba2624de6c9dd5745c788d669dff0562c4b283d447b",
		"registry.connect.redhat.com/perceptilabs/perceptilabs-operator-bundle@sha256:90b5771978da944a76d790e3609d4523b82c1c4b75f86da3960fbf682a697fb3",
		"registry.connect.redhat.com/perceptilabs/perceptilabs-operator-bundle@sha256:30aed432c70f6bd3da4e6365e32da1877affa4b56458c17549b119de17eaa932",
		"registry.connect.redhat.com/perceptilabs/perceptilabs-operator-bundle@sha256:ac9d7ca9b4ca142bd5c983e541cc687fea3143446b79d7d853f7dd654a3001ef",
		"registry.connect.redhat.com/percona/percona-server-mongodb-operator-bundle@sha256:a0a367801ec964b74d015e8c4025930283f8dd5f125776e78a72101d170f0df4",
		"registry.connect.redhat.com/percona/percona-xtradb-cluster-operator-bundle@sha256:a280b29739d410d14616edde83066cad0f63d2e1a191ee1434636e25448d4e56",
		"registry.connect.redhat.com/portshift/operator-bundle@sha256:3165cd171a5e046461a6b55011084ac293e92334983ef65acaf03ec5fc40c141",
		"registry.connect.redhat.com/portshift/operator-bundle@sha256:315706e1ed51c3543b44c0524ee717b36324d128d8dafac2388adf94107c4c48",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:45e1a7f6115be1545cf0ad204a206458277c80e83cff7414726e05f40edb83b5",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:e7981e3a197cf2519d442d180353b489df5f7b180534b064e2f63f5ebf115fe6",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:46948a9049f11cab4b7896b20e29381c8c06b92fbedc2c0605480711f4786323",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:c2bc0d28a5c322d55ad45a82fb1845cf4346b387afc3076e0f891b3d4e6656d3",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:fc7bc43b6a932255f45e1ef8b72f906919fdf222487ccf3227fd1b1840187b5c",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:635efccc15cf37bc8dda8278790cda45596712b378802a3658c09b9435c1c8a7",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:94ceda7ed2ae07580045b723f3c2783a2a454bc6b838ed22edfa54c6f91e8a77",
		"registry.connect.redhat.com/portworx/portworx-certified-bundle@sha256:9d9a3d6259b1b1ef0d48f08aafd9a7fef29ea243742c8ccdc0750fb3ec184e59",
		"registry.connect.redhat.com/starburst/presto-operator-bundle@sha256:75de295cfe227db51aa7a1b1880076175f3e8725682f845b47fa018e2e6791f5",
		"registry.connect.redhat.com/vacava/rapidbiz-operator-bundle@sha256:57e0e6c51ed2b9ebf5347eee1fdd3ebff795cf8cf51f83e43cad4b02ea3872ad",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:2c1cfca71bdbfc3bc58ff026e2e661d2f8a517880b496ab7b781489f5ffe73ab",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:f2916ae447f0b1f969f67bdbc418c17fd61887b31024a06618dbf5426cedaee7",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:539a8f4e847a89523f1acf2e6dfb00ed2776b38ec3f24a5bedb2e71fac65ab5b",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:f4e205936e8fba2fef8e6294c492c7d3c289c9a5e071615abec373d7c5566b85",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:6e6717231634235adfd203b94daa2df8ccc844975d3bcb7a40406080946b15be",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:c75756c5dc30ca757b7d798df0afa12877b62bd2f5602ce360c9144e7038efe3",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:d55a85c57dfd83ad010fead6b3c34cf0d8f098404bfab7dee22bbd66cc993855",
		"registry.connect.redhat.com/rh-marketplace/rh-marketplace-bundle@sha256:50919a4329e9d72ee85cc1370797c0402aa0d6a43924373049a3f5656d4f8538",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:099ce5ed6a2a9a8217676d2901f0872b7aa3d4ada5000b52a44878d5c4bf1cac",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:16da0c69f0e2585be8e44abe6e269dd35176771f5a1ecda3111c8a2f52c790a0",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:292bef653614d21a7d7a8d7df2507978eafcf66910f978b1edf6469a42302874",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:5666acf5598f96dfbfac5704bb70e6578522979ef63b418ed2d1438a32d2440f",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:a16342d44039d3619f08f3fdd411e0074f7809442fbb486939b69f8c15451ecf",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:b0edae8a793cf9e47a9d33a6341e2d2dfd3656ce8f2a4fbaa7d22a7407402b7f",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:9ea442914977e8d29dc1f50a8d87d51363f729e2962a91ae993038b358916996",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:350e0a49321d7a92b95803925f4bcfe97a95d07e1444f3b378fa80370f27087d",
		"registry.connect.redhat.com/redislabs/redis-enterprise-operator-bundle@sha256:889042a1600da27e338f688044802838bb1b8126e9fd900ff0fbccfdf29b75ec",
		"registry.connect.redhat.com/rocketchat/rocketchat-operator-bundle@sha256:939fd27346e704244fda2c6666cda4e6f164d541e89c0000f4a46d91296831c1",
		"registry.connect.redhat.com/ibm/runtime-component-operator-bundle@sha256:8e8792d973514ff1e87acbfbcaef90fa41d7dc66eac939f5e74b307946f7dd37",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:c380d43717df7db52058bcf261ece6ceb0e0ce779b0576d31d25d8a3c9c70e55",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:cb77b47b96bbdc309b17468b8f644bac7206be94125d43a1609f20637937876f",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:1f423936c3134391ba56414c05ece71e90347666f69a2193fe01370b21ce138f",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:0f91afcdf6ef68f7a103834d6bf85938f457af36d3fab2fec164476ecb711cfa",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:64b7bbd99fa305eee77225a1d19c351817eff13bd018a95caf13c06749ff0491",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:01aed6c653b9cd10b942981218ae777c9326a7f18011e52e2fa125beafaa1268",
		"registry.connect.redhat.com/seldonio/seldon-core-operator-bundle@sha256:53569fdf8eecee215755dcd02d56cb906330f84e3bc1015652d9b7a1ddc01edd",
		"registry.connect.redhat.com/sematext/sematext-operator-bundle@sha256:6261bac8dbfc9453bf59eed93a860f55e15529c564058f79246a5bfe9c6ad698",
		"registry.connect.redhat.com/snyk/snyk-operator-bundle@sha256:4830b241f5e18221172ddc3a9916215dcb0164f8ec2b0317cb6d31a7c2ce76ac",
		"registry.connect.redhat.com/splunk/splunk-operator-bundle@sha256:69ab15464e35611399ced93804618510717be529f42c50e70da0ea6a0027a6bb",
		"registry.connect.redhat.com/starburst/starburst-enterprise-helm-operator-bundle@sha256:31b39961bc96d35430fe527f3c7e5fa98c3ecb430e4f8d9d776f1f82e14785cf",
		"registry.connect.redhat.com/stonebranch/stonebranch-operator-bundle@sha256:b39a3421cf285a88e550fd4ad9a875c0e8e0deeb8b893402400682fe576c6bdc",
		"registry.connect.redhat.com/storageos/cluster-operator2-bundle@sha256:d3cf17ee3f76aa8fe6158c276476c9cceb1837d0fbc1cc6c134e1b8194ab1294",
		"registry.connect.redhat.com/blackducksoftware/synopsys-operator-bundle@sha256:46cee91fdddca9dc6d10fadd954676b7d582505418ba0eec0f2788c19ed1a72a",
		"registry.connect.redhat.com/sysdig/sysdig-operator-bundle@sha256:cee924a4b045cb2c5cd0a4cadbe070a47647b03b7c52762c163f5325ca82ed17",
		"registry.connect.redhat.com/turbonomic/t8c-operator-bundle@sha256:e1c153bcadefa513dcb4df736084dcd6885ef014e7ff34dbbbeba8158839c59c",
		"registry.connect.redhat.com/turbonomic/t8c-operator-bundle@sha256:8890c8329b9968a041860d9eea7685a103b6643882dc569eb634d467442c79c2",
		"registry.connect.redhat.com/turbonomic/t8c-operator-bundle@sha256:54455b83daafad874a99acd5da452c19b0f492ed9781b50cc76940059d7673c4",
		"registry.connect.redhat.com/turbonomic/t8c-operator-bundle@sha256:348f3c9216384a33b21fbdfb37e6359f5288cc28381e57e5b0e0e99a18c02b73",
		"registry.connect.redhat.com/turbonomic/t8c-operator-bundle@sha256:d1996f41abc640884dd988174df35b39e4d487bdfe4a1e2c39b1c365d20612e3",
		"registry.connect.redhat.com/atsgen/tf-operator-bundle@sha256:4377e3814893d2071e277b284c633d42586631ef8077a1b31b8d81058fce1380",
		"registry.connect.redhat.com/rhel-timemachine-container/timemachine-operator-bundle@sha256:cf6aab0d357f8dadbbcd596c09cd847752401dd17cbec15270b8213f2fc99dd9",
		"registry.connect.redhat.com/containous/traefikee-operator-bundle@sha256:d8be7fb9621c8a2224d218f62926944a86d704d92db1c2344a0f3724575991b8",
		"registry.connect.redhat.com/ibm/trans-advisor-operator-bundle@sha256:36d0e065b737cfc39015d55c9c71d1aff7d26a1f3a1009fcfbf6534d9a37a4f1",
		"registry.connect.redhat.com/ibm/trans-advisor-operator-bundle@sha256:6de704e756f6859c1090b1904a4d269f90f4a05e7140f183222af5b8815beea2",
		"registry.connect.redhat.com/ibm/trans-advisor-operator-bundle@sha256:ddc9c24a75e0a65b741782c9a51280aecd6a0ae017efafe0d8bb26dd86a848ec",
		"registry.connect.redhat.com/ibm/trans-advisor-operator-bundle@sha256:7a79b29878e35e2e72c774111c7b762db22ba14e8347f6b3dc2dbccbda2c9550",
		"registry.connect.redhat.com/ibm/trans-advisor-operator-bundle@sha256:c50d0c0ab1079d5f079e205c350c8ea2296baa0e0d4444f70bca8fcfa1826f68",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:bc935a0166c56bd3f477cc3027fca6e957856d37a81bb3622afe00898ee5fcfd",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:3a47ed5f905bef037a51e9fb1aabf4e71bdcceedbe6caa7c92352657e3e5ee09",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:99abcf874cce95c5c3980221c322569a7272ab0dfe79f267886153c071f37a2f",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:f5b938c27c07fb9fde2440529040fd6ba52b69ca3482205d87a066f54cf30f22",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:9ac8ad4f8e1b43fe92b34c28ff77ebad503264f790109f0d6dbc65e70a85e72d",
		"registry.connect.redhat.com/ca/uma-operator-bundle@sha256:35b076ab107dd19e7c9e1bd3cd237f66448ebef8fb63d50331d791ba80850bc8",
		"registry.connect.redhat.com/storware/vprotect-operator-bundle@sha256:119f046b27bcf6bc70b1f04d56fe86e8d1c223c8c5ba238f323821a4c14b4bae",
		"registry.connect.redhat.com/storware/vprotect-operator-bundle@sha256:3c8b2d67a329f6cd81666303009bb8a43e1970c5703b93d458f500525585c15b",
		"registry.connect.redhat.com/wavefronthq/wavefront-operator-bundle@sha256:a1cba5d7638a711eaf477dccff47d7a1bb83490c2bb05371a950b8e593bc59f4",
		"registry.connect.redhat.com/zts/xcrypt-operator-1-bundle@sha256:6a9b95934c0b8b3dfa3f71c0052a5e55091c560d8db490bb227f9c4c2219f493",
		"registry.connect.redhat.com/hpestorage/xspc-csi-operator-bundle@sha256:d63dc1c1d2617ab889f43fea4735a5cbe59218d94f5a4138de2b31dd1225ed3a",
		"registry.connect.redhat.com/yugabytedb/yugabyte-operator-bundle@sha256:75af634b49ff92fa9cad7a20a9d6d86689624b15bed4c84889b677c288b8c036",
		"registry.connect.redhat.com/zabbix/zabbixoperator-certified-bundle@sha256:7d7874dc07cdf9fa61a6377e87e836e63629e404415fd8bc98ce3fe88d3932ca",
		"registry.connect.redhat.com/zabbix/zabbixoperator-certified-bundle@sha256:236d7cc772634f20540dc153f9a7fc66357d7d4852962c5c7b5daff61c99e5c8",
		"registry.connect.redhat.com/zadara/zadara-operator-bundle@sha256:a7dc6b758b849f4eab38f38b5bb0c99e951d1ed5c59143337e6c5efb5973f321",
	}

	jsonFinalResult := "cert/bundles_registry_proxy.engineering.redhat.com_rh_osbs_iib@sha256_2bf2a92ad559cbc7a9fff297b4e17dc8a604c23c67fcd052ffd9b6304482552a_2021-09-08.json"

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

	// TODO: We can also try to do the following checks here.
	// Check if we have the head of channel in the deprecate table
	// Check if we have the properties for each head of channel
	
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
