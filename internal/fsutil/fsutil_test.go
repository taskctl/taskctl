package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	type args struct {
		file string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{args: args{file: filepath.Join(cwd, "fsutil.go")}, want: true},
		{args: args{file: filepath.Join(cwd, "fsutil_test.go")}, want: true},
		{args: args{file: filepath.Join(cwd, "manifesto.txt")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileExists(tt.args.file); got != tt.want {
				t.Errorf("FileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMustGetwd(t *testing.T) {
	wd, _ := os.Getwd()
	if wd != MustGetwd() {
		t.Error()
	}
}

func TestMustGetUserHomeDir(t *testing.T) {
	t.Setenv("HOME", "/test")
	hd := MustGetUserHomeDir()
	if hd != "/test" {
		t.Error()
	}
}
