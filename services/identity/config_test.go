package identity

import (
	"context"
	"errors"
	"testing"
)

// synthetic: true — all byte sequences below are non-production test material.
func TestResolveSecretMaterialUsesReferencesOnly(t *testing.T) {
	t.Parallel()
	resolver := &syntheticSecretResolver{values: map[string][]byte{
		OTPKeyRefName:     []byte("synthetic-otp-key-32-bytes-long-0001"),
		SubjectKeyRefName: []byte("synthetic-subject-key-32-bytes-0001"),
		ProofKeyRefName:   []byte("synthetic-proof-key-32-bytes-long-01"),
		SigningKeyRefName: []byte("synthetic-signing-key-32-bytes-0001"),
		RefreshKeyRefName: []byte("synthetic-refresh-key-32-bytes-0001"),
	}}
	material, err := ResolveSecretMaterial(context.Background(), resolver, DefaultSecretReferences())
	if err != nil {
		t.Fatal(err)
	}
	if len(material.OTPKey) < 32 || len(material.SubjectKey) < 32 || len(material.ProofKey) < 32 ||
		len(material.SigningKey) < 32 || len(material.RefreshKey) < 32 || resolver.calls != 5 {
		t.Fatalf("unexpected resolved material lengths/calls: otp=%d subject=%d proof=%d signing=%d refresh=%d calls=%d",
			len(material.OTPKey), len(material.SubjectKey), len(material.ProofKey), len(material.SigningKey), len(material.RefreshKey), resolver.calls)
	}
}

func TestResolveSecretMaterialRejectsRawSecretNamesAndShortValues(t *testing.T) {
	t.Parallel()
	t.Run("raw reference name", func(t *testing.T) {
		resolver := &syntheticSecretResolver{values: map[string][]byte{}}
		refs := DefaultSecretReferences()
		refs.SigningKeyRef = "IDENTITY_SIGNING_KEY"
		if _, err := ResolveSecretMaterial(context.Background(), resolver, refs); err == nil || resolver.calls != 0 {
			t.Fatalf("raw secret name must fail before resolution: err=%v calls=%d", err, resolver.calls)
		}
	})

	t.Run("short resolved value", func(t *testing.T) {
		resolver := &syntheticSecretResolver{values: map[string][]byte{
			OTPKeyRefName:     []byte("short"),
			SubjectKeyRefName: []byte("synthetic-subject-key-32-bytes-0001"),
			ProofKeyRefName:   []byte("synthetic-proof-key-32-bytes-long-01"),
			SigningKeyRefName: []byte("synthetic-signing-key-32-bytes-0001"),
			RefreshKeyRefName: []byte("synthetic-refresh-key-32-bytes-0001"),
		}}
		if _, err := ResolveSecretMaterial(context.Background(), resolver, DefaultSecretReferences()); err == nil {
			t.Fatal("short secret material must fail closed")
		}
	})
}

type syntheticSecretResolver struct {
	values map[string][]byte
	calls  int
}

func (r *syntheticSecretResolver) Resolve(_ context.Context, name string) ([]byte, error) {
	r.calls++
	value, ok := r.values[name]
	if !ok {
		return nil, errors.New("synthetic reference unavailable")
	}
	return append([]byte(nil), value...), nil
}
