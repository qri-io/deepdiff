package difff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatPretty converts a []*Delta into a colored text report, with:
// red "-" for deletions
// green "+" for insertions
// blue "~" for changes (an insert & delete at the same path)
// This is very much a work in progress
func FormatPretty(changes []*Delta) (string, error) {
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
func FormatPrettyColor(changes []*Delta) (string, error) {
	buf := &bytes.Buffer{}
	pretty, err := pretty(changes)
	if err != nil {
		return "", err
	}
	writePrettyString(buf, pretty, 0, true)
	return buf.String(), nil
}

func pretty(changes []*Delta) (pretty map[string]interface{}, err error) {
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
