package envutil

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConvertEnv(t *testing.T) {
	type args struct {
		env map[string]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{args: args{env: map[string]string{"key1": "val1"}}, want: []string{"key1=val1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertEnv(tt.args.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadEnvFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.env")
	content := "KEY1=val1\nKEY2=a=b\n\nNOEQ\nKEY3=val3\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadEnvFile(file)
	if err != nil {
		t.Fatalf("ReadEnvFile() error = %v", err)
	}

	want := map[string]string{
		"KEY1": "val1",
		"KEY2": "a=b", // value containing '=' is preserved
		"KEY3": "val3",
		// "NOEQ" line has no '=' and is skipped, not a panic
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadEnvFile() = %v, want %v", got, want)
	}
}

func TestSanitizeEnviron(t *testing.T) {
	type args struct {
		environ []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "normal var kept",
			args: args{environ: []string{"PATH=/usr/bin"}},
			want: []string{"PATH=/usr/bin"},
		},
		{
			name: "underscore-first kept",
			args: args{environ: []string{"_UNDERSCORE=x"}},
			want: []string{"_UNDERSCORE=x"},
		},
		{
			name: "windows var with parens kept",
			args: args{environ: []string{"ProgramFiles(x86)=C:\\Program Files (x86)"}},
			want: []string{"ProgramFiles(x86)=C:\\Program Files (x86)"},
		},
		{
			name: "cygwin junk dropped",
			args: args{environ: []string{"!::=::\\"}},
			want: []string{},
		},
		{
			name: "windows hidden drive entry dropped (empty key)",
			args: args{environ: []string{"=C:=C:\\foo"}},
			want: []string{},
		},
		{
			name: "no-equals dropped",
			args: args{environ: []string{"NOEQUALS"}},
			want: []string{},
		},
		{
			name: "empty string dropped",
			args: args{environ: []string{""}},
			want: []string{},
		},
		{
			name: "mixed input preserves order and drops invalid entries",
			args: args{environ: []string{
				"PATH=/usr/bin",
				"!::=::\\",
				"=C:=C:\\foo",
				"HOME=/home/user",
				"NOEQUALS",
				"_OK=1",
				"",
			}},
			want: []string{"PATH=/usr/bin", "HOME=/home/user", "_OK=1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeEnviron(tt.args.environ); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SanitizeEnviron() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadEnvFile_notExist(t *testing.T) {
	if _, err := ReadEnvFile(filepath.Join(t.TempDir(), "missing.env")); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
