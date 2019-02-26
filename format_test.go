package difff

import "testing"

func TestFormatPretty(t *testing.T) {
	patch := []*Delta{
		{Type: DTInsert, Path: "/a", Value: 5},
		{Type: DTUpdate, Path: "/a", Value: 5},
		{Type: DTDelete, Path: "/a", Value: 5},
		{Type: DTMove, Path: "/a", Value: 5},
	}

	str, err := FormatPretty(patch)
	if err != nil {
		t.Fatal(err)
	}
	// TODO (b5) = need to actually tests this stuff
	t.Log(str)
}
