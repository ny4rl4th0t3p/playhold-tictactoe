package main

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		list    string
		want    []netip.Prefix
		wantErr bool
	}{
		{name: "none", list: ""},
		{name: "one address", list: "172.30.0.2/32", want: []netip.Prefix{netip.MustParsePrefix("172.30.0.2/32")}},
		{
			name: "two ranges",
			list: "10.0.0.0/8,172.30.0.0/24",
			want: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("172.30.0.0/24")},
		},
		{name: "an address without its length", list: "172.30.0.2", wantErr: true},
		{name: "not an address", list: "proxy", wantErr: true},
		{name: "a space after the comma", list: "10.0.0.0/8, 172.30.0.0/24", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := trustedProxies(tt.list)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
