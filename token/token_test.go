package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenFromBearerString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid bearer token",
			input:   "Bearer abc123",
			want:    "abc123",
			wantErr: false,
		},
		{
			name:    "Valid bearer token with spaces",
			input:   "Bearer   xyz789   ",
			want:    "xyz789",
			wantErr: false,
		},
		{
			name:    "Missing 'Bearer' prefix",
			input:   "abc123",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Only 'Bearer' without token",
			input:   "Bearer ",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Lowercase 'bearer'",
			input:   "bearer abc123",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Token with spaces",
			input:   "Bearer abc 123 xyz",
			want:    "abc 123 xyz",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromBearerString(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
