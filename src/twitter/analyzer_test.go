package twitter

import (
	"testing"
)

func TestProcessDirectUserInteractions(t *testing.T) {

	tests := []struct {
		name          string
		userId        string
		expectResult  UserInteractions
		inputResponse *UserTweetsResponse
		inputResult   *UserInteractions
	}{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ProcessDirectUserInteractions(tt.inputResponse, tt.inputResult)
		})
	}
}
