package deepdiff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatPretty converts a Deltas into a colored text report, with:
// red "-" for deletions
// green "+" for insertions
// blue "~" for changes (an insert & delete at the same path)
// This is very much a work in progress
func FormatPretty(changes Deltas) (string, error) {
	buf := &bytes.Buffer{}
	pretty, err := pretty(changes)
	if err != nil {
		return "", err
	}
	writePrettyString(buf, pretty, 0, false)
	return buf.String(), nil
}

// FormatPrettyColor is the same as format pretty, but with tty color tags
// to print colored text to terminals
func FormatPrettyColor(changes Deltas) (string, error) {
	buf := &bytes.Buffer{}
	pretty, err := pretty(changes)
	if err != nil {
		return "", err
	}
	writePrettyString(buf, pretty, 0, true)
	return buf.String(), nil
}

func pretty(changes Deltas) (pretty map[string]interface{}, err error) {
	pretty = map[string]interface{}{}
	var data []byte
	for _, diff := range changes {

		path := strings.Split(diff.Path, "/")
		name := ""
		el := pretty
		for i, p := range path {
			name = p
			if i < len(path)-1 && name != "" {
				if el[p] == nil {
					el[p] = map[string]interface{}{}
				}
				el = el[p].(map[string]interface{})
			}
		}

		switch diff.Type {
		case DTInsert:
			if data, err = json.Marshal(diff.Value); err != nil {
				return
			}
			el["+ "+name] = string(data)
		case DTDelete:
			if data, err = json.Marshal(diff.Value); err != nil {
				return
			}
			el["- "+name] = string(data)
		case DTUpdate:
			if data, err = json.Marshal(diff.Value); err != nil {
				return
			}
			el["~ "+name] = string(data)
		}
	}
	return
}

func writePrettyString(buf *bytes.Buffer, pretty map[string]interface{}, indent int, color bool) {
	var (
		keys                                              = make([]string, len(pretty))
		insertColor, deleteColor, updateColor, closeColor string
	)

	if color {
		insertColor = "\x1b[32m"
		deleteColor = "\x1b[31m"
		updateColor = "\x1b[34m"
		closeColor = "\x1b[0m"
	}

	i := 0
	for key := range pretty {
		keys[i] = key
		i++
	}
	sort.Strings(keys)

	for _, key := range keys {
		switch val := pretty[key].(type) {
		case map[string]interface{}:
			buf.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("  ", indent), key))
			writePrettyString(buf, val, indent+1, color)
		case string:
			switch key[0] {
			case '+':
				buf.WriteString(fmt.Sprintf("%s%s", strings.Repeat("  ", indent), insertColor))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString(fmt.Sprintf("%s\n", closeColor))
			case '-':
				buf.WriteString(fmt.Sprintf("%s%s", strings.Repeat("  ", indent), deleteColor))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString(fmt.Sprintf("%s\n", closeColor))
			case '~':
				buf.WriteString(fmt.Sprintf("%s%s", strings.Repeat("  ", indent), updateColor))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString(fmt.Sprintf("%s\n", closeColor))
			}
		}
	}
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
