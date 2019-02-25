package difff

import "testing"

func TestFormatPretty(t *testing.T) {
	patch := []*Delta{
		{Type: DTInsert, DstPath: "/a", DstVal: 5},
		{Type: DTUpdate, DstPath: "/a", DstVal: 5},
		{Type: DTDelete, DstPath: "/a", DstVal: 5},
		{Type: DTMove, DstPath: "/a", DstVal: 5},
	}

	str, err := FormatPretty(patch)
	if err != nil {
		t.Fatal(err)
	}
	// TODO (b5) = need to actually tests this stuff
	t.Log(str)
}
