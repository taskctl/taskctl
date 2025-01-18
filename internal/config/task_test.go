package config

import (
	"testing"

	"github.com/taskctl/taskctl/variables"
)

func Test_buildTask(t *testing.T) {
	type args struct {
		def *taskDefinition
	}
	tests := []struct {
		name    string
		args    args
		want    variables.Container
		wantErr bool
	}{
		{args: args{def: &taskDefinition{
			EnvFile: "testdata/.env",
		}}, want: variables.FromMap(map[string]string{"VAR_1": "VAL_1_2", "VAR_2": "VAL_2"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildTask(tt.args.def, &loaderContext{})
			if (err != nil) != tt.wantErr {
				t.Errorf("buildTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for k, v := range tt.want.Map() {
				if got.Env.Get(k) != v {
					t.Errorf("buildTask() env error, want %s, got %s", v, got.Env.Get(k))
				}
			}
		})
	}
}
