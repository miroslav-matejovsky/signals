package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGreeting(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "world", want: "Hello, world!"},
		{name: "Go", want: "Hello, Go!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, greeting(tt.name))
		})
	}
}
