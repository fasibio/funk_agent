package main

import "testing"

func Test_getFilledValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{
			name:     "Get Value",
			value:    "value",
			fallback: "ignored",
			want:     "value",
		},
		{
			name:     "Get Fallback",
			value:    "",
			fallback: "fallback",
			want:     "fallback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getFilledValue(tt.value, tt.fallback); got != tt.want {
				t.Errorf("getFilledValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
