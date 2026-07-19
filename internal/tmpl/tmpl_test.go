package tmpl

import "testing"

func TestRenderString(t *testing.T) {
	type args struct {
		tmpl      string
		variables map[string]any
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{args: args{tmpl: "hello, {{ .Name }}!", variables: map[string]any{"Name": "world"}}, want: "hello, world!"},
		{args: args{tmpl: "hello, {{ .Name | default \"John\" }}!", variables: map[string]any{"Name": ""}}, want: "hello, John!"},
		{args: args{tmpl: "hello, {{ .Name }}!", variables: make(map[string]any)}, wantErr: true},
		{args: args{tmpl: "hello, {{ .Name", variables: make(map[string]any)}, wantErr: true},
		{args: args{tmpl: "{{ .Task.Name }}", variables: map[string]any{"Task": map[string]any{"Name": "t1"}}}, want: "t1"},
		{args: args{tmpl: "{{ .Tasks.Build.Stdout }}", variables: map[string]any{"Tasks": map[string]any{"Build": map[string]any{"Stdout": "out"}}}}, want: "out"},
		{args: args{tmpl: "{{ .Task.Missing }}", variables: map[string]any{"Task": map[string]any{"Name": "t1"}}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderString(tt.args.tmpl, tt.args.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("RenderString() got = %v, want %v", got, tt.want)
			}
		})
	}
}
