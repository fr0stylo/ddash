package servicecatalog

import "testing"

func TestParseDependencyInputs(t *testing.T) {
	values := ParseDependencyInputs("orders", " billing, auth;orders,AUTH\nedge ")
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}
	if values[0] != "billing" || values[1] != "auth" || values[2] != "edge" {
		t.Fatalf("unexpected values: %+v", values)
	}
}
