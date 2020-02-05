package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		url       string
		opts      []QueryOption
		expectErr error
		expectRet string
	}{
		{
			url:       "github.com",
			opts:      nil,
			expectErr: nil,
			expectRet: "github.com",
		},
		{
			url:       "https://github.com",
			opts:      nil,
			expectErr: nil,
			expectRet: "https://github.com",
		},
		{
			url: "https://github.com",
			opts: []QueryOption{
				QueryOptionSort("createdAt"),
				QueryOptionOrder("asc"),
				QueryOptionPage(1),
				QueryOptionPerPage(50),
			},
			expectErr: nil,
			expectRet: "https://github.com?order=asc&page=1&per_page=50&sort=createdAt",
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ret, err := buildURL(tt.url, tt.opts...)
			assert.Equal(t, tt.expectErr, err)
			assert.Equal(t, tt.expectRet, ret)
		})
	}
}
