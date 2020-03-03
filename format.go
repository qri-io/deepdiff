package deepdiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// FormatPrettyString is a convenice wrapper that outputs to a string instead of
// an io.Writer
func FormatPrettyString(changes Deltas, colorTTY bool) (string, error) {
	buf := &bytes.Buffer{}
	if err := FormatPretty(buf, changes, colorTTY); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FormatPretty writes a text report to w. if colorTTY is true it will add
// red "-" for deletions
// green "+" for insertions
// blue "~" for changes (an insert & delete at the same path)
// This is very much a work in progress
func FormatPretty(w io.Writer, changes Deltas, colorTTY bool) error {
	var colorMap map[Operation]string

	if colorTTY {
		colorMap = map[Operation]string{
			Operation("close"): "\x1b[0m", // end color tag

			DTContext: "\x1b[37m", // netural
			DTInsert:  "\x1b[32m", // green
			DTDelete:  "\x1b[31m", // red
			DTUpdate:  "\x1b[34m", // blue
		}
	}

	return formatPretty(w, changes, 0, colorMap)
}

func formatPretty(w io.Writer, changes Deltas, indent int, colorMap map[Operation]string) error {
	for _, d := range changes {
		dataStr := ""
		if d.Value != nil {
			d, err := json.Marshal(d.Value)
			if err != nil {
				return err
			}
			dataStr = string(d)
		}
		fmt.Fprintf(w, "%s%s%s%s: %s%s\n", strings.Repeat("  ", indent), colorMap[d.Type], d.Type, d.Path, dataStr, colorMap[Operation("close")])
		if len(d.Deltas) > 0 {
			if err := formatPretty(w, d.Deltas, indent+1, colorMap); err != nil {
				return err
			}
		}
	}

	return nil
}

// FormatPrettyStatsString prints a string of stats info
func FormatPrettyStatsString(diffStat *Stats, colorTTY bool) string {
	buf := &bytes.Buffer{}
	FormatPrettyStats(buf, diffStat, colorTTY)
	return buf.String()
}

// FormatPrettyStats writes stats info to a supplied writer destination,
// optionally adding terminal color tags
func FormatPrettyStats(w io.Writer, diffStat *Stats, colorTTY bool) {
	var colorMap map[Operation]string

	if colorTTY {
		colorMap = map[Operation]string{
			Operation("close"): "\x1b[0m", // end color tag

			DTContext: "\x1b[37m", // netural
			DTInsert:  "\x1b[32m", // green
			DTDelete:  "\x1b[31m", // red
			DTUpdate:  "\x1b[34m", // blue
		}
	}

	formatStats(w, diffStat, colorMap)
}

func formatStats(w io.Writer, ds *Stats, colorMap map[Operation]string) {
	if ds == nil {
		return
	}
	closeColor := colorMap[Operation("close")]

	elsColor := colorMap[DTInsert]
	change := ds.NodeChange()
	elementsWord := "elements"
	sign := "+"
	if change < 0 {
		elsColor = colorMap[DTDelete]
		sign = ""
	} else if change == 0 {
		elsColor = colorMap[DTContext]
		sign = ""
	}
	if change == 1 || change == -1 {
		elementsWord = "element"
	}

	fmt.Fprintf(w, "%s%s%d %s%s%s%s.",
		elsColor, sign, change, closeColor,
		colorMap[DTContext], elementsWord, closeColor,
	)

	insertsWord := "inserts"
	if ds.Inserts == 1 {
		insertsWord = "insert"
	}
	fmt.Fprintf(w, " %s%d %s.%s", colorMap[DTInsert], ds.Inserts, insertsWord, closeColor)

	deletesWord := "deletes"
	if ds.Deletes == 1 {
		deletesWord = "delete"
	}
	fmt.Fprintf(w, " %s%d %s.%s", colorMap[DTDelete], ds.Deletes, deletesWord, closeColor)

	if ds.Updates > 0 {
		updatesWord := "updates"
		if ds.Updates == 1 {
			updatesWord = "update"
		}
		fmt.Fprintf(w, " %s%d %s.%s", colorMap[DTUpdate], ds.Updates, updatesWord, closeColor)
	}
	fmt.Fprintf(w, "\n")
}
