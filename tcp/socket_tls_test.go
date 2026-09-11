package tcp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const exampleHost = "example.com"

// TestTLSField verifies that the TLS field in connectOptions can be set.
func TestTLSField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts connectOptions
		want bool
	}{
		{
			name: "TLS enabled",
			opts: connectOptions{
				Host: exampleHost,
				Port: 443,
				TLS:  true,
			},
			want: true,
		},
		{
			name: "TLS disabled",
			opts: connectOptions{
				Host: exampleHost,
				Port: 80,
				TLS:  false,
			},
			want: false,
		},
		{
			name: "TLS default (false)",
			opts: connectOptions{
				Host: exampleHost,
				Port: 80,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, tt.opts.TLS)
		})
	}
}
