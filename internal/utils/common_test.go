package utils

import "testing"

func TestFormatPercentage(t *testing.T) {
	type args struct {
		value float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "zero value",
			args: args{
				value: 0,
			},
			want: "+0.0%",
		},
		{
			name: "positive value",
			args: args{
				value: 1,
			},
			want: "+1.0%",
		},
		{
			name: "negative value",
			args: args{
				value: -1,
			},
			want: "-1.0%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatPercentage(tt.args.value); got != tt.want {
				t.Errorf("FormatPercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}
