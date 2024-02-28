package twitter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name           string
		inputUserId    string
		inputReferrals []string
		wantOutput     string
	}{
		{
			name:           "single referral",
			inputUserId:    "user-1-id",
			inputReferrals: []string{"referral-1-id"},
			wantOutput:     "(from:user-1-id to:referral-1-id) OR (from:referral-1-id to:user-1-id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildSearchQuery(tt.inputUserId, tt.inputReferrals)
			assert.Equal(t, tt.wantOutput, query)
		})
	}

}
