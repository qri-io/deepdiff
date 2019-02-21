package difff

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatPrettyJSON converts a []*Delta into a colored text report
func FormatPrettyJSON(changes []*Delta) (string, error) {
	pretty := map[string]interface{}{}
	for _, diff := range changes {

		path := strings.Split(diff.DstPath, "/")
		if diff.Type == DTDelete {
			path = strings.Split(diff.SrcPath, "/")
		}
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
			data, err := json.Marshal(diff.DstVal)
			if err != nil {
				return "", err
			}
			el["+ "+name] = string(data)
		case DTDelete:
			data, err := json.Marshal(diff.SrcVal)
			if err != nil {
				return "", err
			}
			el["- "+name] = string(data)
		case DTUpdate:
			data, err := json.Marshal(diff.DstVal)
			if err != nil {
				return "", err
			}
			el["~ "+name] = string(data)
		}
	}

	buf := &bytes.Buffer{}
	writePrettyJSONString(buf, pretty, 0)
	return buf.String(), nil
}

func writePrettyJSONString(buf *bytes.Buffer, pretty map[string]interface{}, indent int) {
	var keys = make([]string, len(pretty))
	i := 0
	for key := range pretty {
		keys[i] = key
		i++
	}

	for _, key := range sort.StringSlice(keys) {
		switch val := pretty[key].(type) {
		case map[string]interface{}:
			buf.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("  ", indent), key))
			writePrettyJSONString(buf, val, indent+1)
		case string:
			switch key[0] {
			case '+':
				buf.WriteString(fmt.Sprintf("%s\x1b[32m", strings.Repeat("  ", indent)))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString("\x1b[0m\n")
			case '-':
				buf.WriteString(fmt.Sprintf("%s\x1b[31m", strings.Repeat("  ", indent)))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString("\x1b[0m\n")
			case '~':
				buf.WriteString(fmt.Sprintf("%s\x1b[34m", strings.Repeat("  ", indent)))
				buf.WriteString(fmt.Sprintf("%s: %s", key, pretty[key]))
				buf.WriteString("\x1b[0m\n")
			}
		}
	}
}

// FormatPrettyText converts a []*Delta into a colored text report
func FormatPrettyText(changes []*Delta) (string, error) {
	var buf bytes.Buffer

	for _, diff := range changes {
		var text string

		switch diff.Type {
		case DTInsert:
			if data, err := json.Marshal(diff.DstVal); err == nil {
				text = string(data)
			}
			buf.WriteString(diff.DstPath)
			buf.WriteString("  \x1b[32m")
			buf.WriteString(text)
			buf.WriteString("\x1b[0m\n")
		case DTDelete:
			if data, err := json.Marshal(diff.SrcVal); err == nil {
				text = string(data)
			}
			buf.WriteString(diff.SrcPath)
			buf.WriteString("  \x1b[31m")
			buf.WriteString(text)
			buf.WriteString("\x1b[0m\n")
		case DTUpdate:
			if data, err := json.Marshal(diff.DstVal); err == nil {
				text = string(data)
			}
			buf.WriteString(diff.DstPath)
			buf.WriteString("  \x1b[34m")
			buf.WriteString(text)
			buf.WriteString("\x1b[0m\n")
		}
	}

	return buf.String(), nil
}
