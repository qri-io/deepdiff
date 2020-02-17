package deepdiff

import "testing"

func TestFormatPretty(t *testing.T) {
	patch := Deltas{
		{Type: DTInsert, Path: "a", Value: 5},
		{Type: DTUpdate, Path: "a", Value: 5},
		{Type: DTDelete, Path: "a", Value: 5},
		{Type: DTMove, Path: "a", Value: 5},
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
			&Stats{Left: 2, Right: 6, Inserts: 6, Updates: 2, Deletes: 2, Moves: 2},
			"+4 elements. 6 inserts. 2 deletes. 2 updates. 2 moves.\n",
		},
		{"all singular",
			&Stats{Left: 2, Right: 1, Inserts: 1, Updates: 1, Deletes: 1, Moves: 1},
			"-1 element. 1 insert. 1 delete. 1 update. 1 move.\n",
		},
	}

	for i, c := range cases {
		got := FormatPrettyStats(c.input)
		if got != c.expect {
			t.Errorf("%d %s\nwant:\n%s\ngot:\n%s", i, c.description, c.expect, got)
		}
	}
}

func TestFormatStatsNull(t *testing.T) {
	got := FormatPrettyStats(nil)
	expect := `<nil>`
	if got != expect {
		t.Errorf("want:\n%s\ngot:\n%s", expect, got)
	}
}
