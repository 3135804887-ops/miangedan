package consent

import "context"

// Observation is a content-free, low-cardinality consent signal. Production
// composition maps one record to an OpenTelemetry metric, span event and
// redacted structured log. User IDs, scope hashes, request keys and evidence
// are deliberately absent.
type Observation struct {
	Operation       string
	Outcome         string
	ConsentType     Type
	EffectiveStatus EffectiveStatus
	ErrorCode       Code
	DataRegion      string
}

// Observer receives sanitized signals without participating in authorization.
type Observer interface {
	Record(context.Context, Observation)
}

// NoopObserver keeps business behavior independent from telemetry availability.
type NoopObserver struct{}

// Record implements Observer.
func (NoopObserver) Record(context.Context, Observation) {}
