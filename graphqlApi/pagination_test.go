package graphqlApi

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestDecodeCursor(t *testing.T) {
	// base64 encoded {"offset": 1}
	encoded_cursor := "eyJvZmZzZXQiOiAxfQ=="
	data, err := DecodeCursor(encoded_cursor)
	if err != nil {
		t.Fatalf("%s", err)
	}

	offset := 1
	expected := CursorData{Offset: &offset}
	if !cmp.Equal(data, expected) {
		t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(data, expected))
	}
}
