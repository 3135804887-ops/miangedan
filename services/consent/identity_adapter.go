package consent

import (
	"context"
	"errors"

	"miangedan/services/identity"
)

// IdentityAgeReader is the least-privilege identity view needed for raw
// recording eligibility. It returns an enum only, never profile or contact data.
type IdentityAgeReader interface {
	GetAgeStatus(context.Context, string, string) (identity.AgeStatus, error)
}

// IdentityRecordingEligibility adapts TASK-010 age state to TASK-011. Only an
// adult is eligible; guardian-verified and pending minors remain ineligible.
type IdentityRecordingEligibility struct {
	reader IdentityAgeReader
}

// NewIdentityRecordingEligibility creates a fail-closed identity policy adapter.
func NewIdentityRecordingEligibility(reader IdentityAgeReader) (*IdentityRecordingEligibility, error) {
	if reader == nil {
		return nil, errors.New("identity age reader is required")
	}
	return &IdentityRecordingEligibility{reader: reader}, nil
}

// AllowRawAV implements RecordingEligibility.
func (a *IdentityRecordingEligibility) AllowRawAV(
	ctx context.Context,
	userID string,
	dataRegion string,
) (bool, error) {
	status, err := a.reader.GetAgeStatus(ctx, userID, dataRegion)
	if err != nil {
		return false, err
	}
	if !status.Valid() {
		return false, errors.New("identity returned invalid age status")
	}
	return status == identity.AgeAdult, nil
}
