package utils

import "testing"

func TestIsURL(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "HTTP URL", args: args{s: "http://github.com/"}, want: true},
		{name: "HTTPS URL", args: args{s: "https://github.com/"}, want: true},
		{name: "Windows path", args: args{s: "C:\\Windows"}, want: false},
		{name: "Mailto", args: args{s: "mailto:admin@example.org"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsURL(tt.args.s); got != tt.want {
				t.Errorf("IsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
