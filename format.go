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

// FormatPrettyStats prints a string of stats info
func FormatPrettyStats(diffStat *Stats) string {
	return formatStats(diffStat, false)
}

// FormatPrettyStatsColor prints a string of stats info with ANSI colors
func FormatPrettyStatsColor(diffStat *Stats) string {
	return formatStats(diffStat, true)
}

func formatStats(ds *Stats, color bool) string {
	var (
		neutralColor, insertColor, deleteColor, updateColor, closeColor string
	)

	if ds == nil {
		return "<nil>"
	}

	if color {
		neutralColor = "\x1b[37m"
		insertColor = "\x1b[32m"
		deleteColor = "\x1b[31m"
		updateColor = "\x1b[34m"
		closeColor = "\x1b[0m"
	}

	buf := &bytes.Buffer{}

	elsColor := insertColor
	change := ds.NodeChange()
	elementsWord := "elements"
	sign := "+"
	if change < 0 {
		elsColor = deleteColor
		sign = ""
	} else if change == 0 {
		elsColor = neutralColor
		sign = ""
	}
	if change == 1 || change == -1 {
		elementsWord = "element"
	}

	buf.WriteString(fmt.Sprintf("%s%s%d %s%s%s%s.",
		elsColor, sign, change, closeColor,
		neutralColor, elementsWord, closeColor,
	))

	insertsWord := "inserts"
	if ds.Inserts == 1 {
		insertsWord = "insert"
	}
	buf.WriteString(fmt.Sprintf(" %s%d %s.%s", insertColor, ds.Inserts, insertsWord, closeColor))

	deletesWord := "deletes"
	if ds.Deletes == 1 {
		deletesWord = "delete"
	}
	buf.WriteString(fmt.Sprintf(" %s%d %s.%s", deleteColor, ds.Deletes, deletesWord, closeColor))

	updatesWord := "updates"
	if ds.Updates == 1 {
		updatesWord = "update"
	}
	buf.WriteString(fmt.Sprintf(" %s%d %s.%s", updateColor, ds.Updates, updatesWord, closeColor))

	if ds.Moves > 0 {
		movesWord := "moves"
		if ds.Moves == 1 {
			movesWord = "move"
		}
		buf.WriteString(fmt.Sprintf(" %s%d %s.%s", updateColor, ds.Moves, movesWord, closeColor))
	}

	buf.WriteRune('\n')

	return buf.String()
}
