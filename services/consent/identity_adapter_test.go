package consent

import (
	"context"
	"errors"
	"testing"

	"miangedan/services/identity"
)

// synthetic: true — this adapter test uses enum-only, non-personal fixtures.
type syntheticAgeReader struct {
	status identity.AgeStatus
	err    error
}

func (r syntheticAgeReader) GetAgeStatus(context.Context, string, string) (identity.AgeStatus, error) {
	return r.status, r.err
}

func TestIdentityRecordingEligibilityFailClosed(t *testing.T) {
	tests := []struct {
		name    string
		reader  syntheticAgeReader
		allowed bool
		wantErr bool
	}{
		{"adult", syntheticAgeReader{status: identity.AgeAdult}, true, false},
		{"guardian verified minor", syntheticAgeReader{status: identity.AgeMinorGuardianVerified}, false, false},
		{"pending minor", syntheticAgeReader{status: identity.AgeMinorPending}, false, false},
		{"invalid status", syntheticAgeReader{status: identity.AgeStatus("synthetic_invalid")}, false, true},
		{"identity unavailable", syntheticAgeReader{err: errors.New("synthetic identity outage")}, false, true},
	}
	for _, row := range tests {
		t.Run(row.name, func(t *testing.T) {
			adapter, err := NewIdentityRecordingEligibility(row.reader)
			if err != nil {
				t.Fatal(err)
			}
			allowed, err := adapter.AllowRawAV(context.Background(), syntheticUserID, "intl")
			if allowed != row.allowed || (err != nil) != row.wantErr {
				t.Fatalf("allowed=%v err=%v", allowed, err)
			}
		})
	}
	if _, err := NewIdentityRecordingEligibility(nil); err == nil {
		t.Fatal("nil identity age reader must fail")
	}
}
