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
	"testing"
)

func TestBuildQuery(t *testing.T) {
	type args struct {
		report Data
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should build only the select when has not flags values",
			args: args{report: Data{Flags: BindFlags{}}},
			want: "SELECT o.name, o.csv, o.bundlepath FROM operatorbundle o",
		},
		{
			name: "should build sql for head only",
			args: args{report: Data{Flags: BindFlags{HeadOnly: true}}},
			want: "SELECT o.name, o.csv, o.bundlepath FROM operatorbundle o, channel c WHERE c.head_operatorbundle_name == o.name",
		},
		{
			name: "should build sql for head only with limit",
			args: args{report: Data{Flags: BindFlags{
				IndexImage: "registry.redhat.io/redhat/redhat-operator-index:v4.7",
				HeadOnly:   true,
				OutputPath: "../testdata/xls",
				Limit:      int32(3),
			}}},
			want: "SELECT o.name, o.csv, o.bundlepath FROM operatorbundle o, channel c WHERE c.head_operatorbundle_name == o.name LIMIT 3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.report.BuildBundlesQuery()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildBundlesQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildBundlesQuery() got = %v, want %v", got, tt.want)
			}
		})
	}
}
