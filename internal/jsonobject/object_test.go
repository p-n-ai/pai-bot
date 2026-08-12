package jsonobject

import (
	"encoding/json"
	"testing"
)

func TestObjectRoundTrip(t *testing.T) {
	object := New(
		Member("name", "lesson"),
		Member("count", 3),
	)

	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	name, found, err := Get[string](parsed, "name")
	if err != nil {
		t.Fatal(err)
	}
	if !found || name != "lesson" {
		t.Fatalf("name = %q, found = %t", name, found)
	}
}

func TestObjectRejectsNonObject(t *testing.T) {
	if _, err := Parse([]byte(`[]`)); err == nil {
		t.Fatal("expected non-object input to fail")
	}
}
