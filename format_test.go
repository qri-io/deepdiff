package deepdiff

import "testing"

func TestFormatPretty(t *testing.T) {
	patch := Deltas{
		{Type: DTInsert, Path: StringAddr("a"), Value: 5},
		{Type: DTUpdate, Path: StringAddr("a"), Value: 5},
		{Type: DTDelete, Path: StringAddr("a"), Value: 5},
	}

	str, err := FormatPrettyString(patch, false)
	if err != nil {
		t.Fatal(err)
	}
	// TODO (b5) = need to actually tests this stuff
	t.Log(str)
}

func TestFormatStatsPretty(t *testing.T) {
	cases := []struct {
		description string
		input       *Stats
		expect      string
	}{
		{"all plural",
			&Stats{Left: 2, Right: 6, Inserts: 6, Updates: 2, Deletes: 2},
			"+4 elements. 6 inserts. 2 deletes. 2 updates.\n",
		},
		{"all singular",
			&Stats{Left: 2, Right: 1, Inserts: 1, Updates: 1, Deletes: 1},
			"-1 element. 1 insert. 1 delete. 1 update.\n",
		},
	}

	for i, c := range cases {
		got := FormatPrettyStatsString(c.input, false)
		if got != c.expect {
			t.Errorf("%d %s\nwant:\n%s\ngot:\n%s", i, c.description, c.expect, got)
		}
	}
}

func TestFormatStatsNull(t *testing.T) {
	got := FormatPrettyStatsString(nil, false)
	expect := ``
	if got != expect {
		t.Errorf("want:\n%s\ngot:\n%s", expect, got)
	}
}
