package model

import (
	"testing"
)

func TestPGStringToTags(t *testing.T) {
	testTagsString := `{"tag1", "tag2 with space", "tag3", "tag4 with space"}`
	expectedOutput := TagsType{"tag1", "tag2 with space", "tag3", "tag4 with space"}

	x := PGStringToTags(testTagsString)

	if len(x) != len(expectedOutput) {
		t.Fatalf("Parsed output is incorrect length: %d != %d", len(x), len(expectedOutput))
	}

	for i := range x {
		if x[i] != expectedOutput[i] {
			t.Fatalf("Parsed output element #%d does not match expected output: %s != %s", i, x[i], expectedOutput[i])
		}
	}
}

func TestStringString(t *testing.T) {
	testTags := TagsType{"tag1", "tag2 with space", "tag3", "tag4 with space"}
	expectedOutput := "tag1, tag2 with space, tag3, tag4 with space"

	if output := testTags.String(); output != expectedOutput {
		t.Fatalf("String() does not match expected output: %s != %s", output, expectedOutput)
	}
}
