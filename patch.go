package difff

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Patch applies a change script (patch) to a value
func Patch(v interface{}, patch []*Delta) (err error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("passed in value must be a pointer")
	}

	for i, dlt := range patch {
		if err := applyDelta(rv.Elem(), dlt); err != nil {
			return fmt.Errorf("patch %d: %s", i, err)
		}
	}
	return nil
}

func applyDelta(tree reflect.Value, dlt *Delta) error {
	switch dlt.Type {
	case DTUpdate:
		return updateValue(tree, dlt.Path, dlt.Value)
	case DTInsert:
		return insertValue(tree, dlt.Path, dlt.Value)
	case DTDelete:
		return deleteValue(tree, dlt.Path)
	case DTMove:
		return moveValue(tree, dlt.SourcePath, dlt.Path, dlt.Value)
	default:
		return fmt.Errorf("unknown delta type")
	}
}

func updateValue(tree reflect.Value, path string, set interface{}) error {
	parent, name, err := pathToParent(tree, path)
	if err != nil {
		return err
	}

	val := parent
	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		val.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(set))
	case reflect.Slice:
		i, err := strconv.Atoi(name)
		if err != nil {
			return err
		}
		val.Index(i).Set(reflect.ValueOf(set))
	default:
		return fmt.Errorf("unrecognized data type for path '%s': %T", path, parent)
	}

	return nil
}

func deleteValue(tree reflect.Value, path string) error {
	parent, name, err := pathToParent(tree, path)
	if err != nil {
		return err
	}

	val := parent
	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		val.SetMapIndex(reflect.ValueOf(name), reflect.Value{})
	case reflect.Slice:
		i, err := strconv.Atoi(name)
		if err != nil {
			return err
		}
		l := val.Len()
		if i >= l {
			return fmt.Errorf("array index %d exceeds %d at path %s", i, l, path)
		} else if i == l-1 {
			parent.Set(val.Slice(0, l-1))
		} else {
			parent.Set(reflect.AppendSlice(val.Slice(0, i), val.Slice(i+1, l)))
		}
	default:
		return fmt.Errorf("unrecognized data type for path '%s': %T", path, parent)
	}

	return nil
}

func insertValue(tree reflect.Value, path string, insert interface{}) error {
	parent, name, err := pathToParent(tree, path)
	if err != nil {
		return err
	}

	val := parent
	if val.Kind() == reflect.Interface {
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Map:
		val.SetMapIndex(reflect.ValueOf(name), reflect.ValueOf(insert))
	case reflect.Slice:
		i, err := strconv.Atoi(name)
		if err != nil {
			return err
		}
		l := val.Len()
		if i > l {
			return fmt.Errorf("array index %d exceeds %d at path %s", i, l, path)
		} else if i == l {
			parent.Set(reflect.Append(val, reflect.ValueOf(insert)))
		} else {
			// TODO (b5): reflect.Append seems to make descructive edits to val if used directly
			// so we need to copy before append. really need to find a concise way to say the following
			// using reflection:
			// parent = append(append(val[i:], elem), val[i:]...)
			slcp := reflect.MakeSlice(val.Type(), i, i)
			reflect.Copy(slcp, val.Slice(0, i))
			parent.Set(
				reflect.AppendSlice(
					reflect.Append(slcp, reflect.ValueOf(insert)),
					val.Slice(i, l),
				),
			)
		}
	default:
		return fmt.Errorf("unrecognized data type for path '%s': %T", path, parent)
	}

	return nil
}

func moveValue(tree reflect.Value, from, to string, val interface{}) error {
	if err := deleteValue(tree, from); err != nil {
		return err
	}
	return insertValue(tree, to, val)
}

func pathToParent(tree reflect.Value, path string) (reflect.Value, string, error) {
	components := strings.Split(path, "/")
	if len(components) < 1 {
		return tree, "", fmt.Errorf("invalid path: %s", path)
	}
	if components[0] == "" && len(components) > 1 {
		components = components[1:]
	}

	elem := tree
	for len(components) > 1 {
		sel := components[0]
		derefed := elem
		if elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
			derefed = elem.Elem()
		}

		switch derefed.Kind() {
		// case reflect.Struct:
		// 	elem = elem.FieldByNameFunc(func(str string) bool {
		// 		return strings.ToLower(str) == sel
		// 	})
		case reflect.Slice:
			index, err := strconv.Atoi(sel)
			if err != nil {
				return elem, sel, fmt.Errorf("invalid index value: %s", sel)
			}
			elem = derefed.Index(index)
		case reflect.Map:
			found := false
			for _, key := range derefed.MapKeys() {
				if key.String() == sel {
					// elem = elem.MapIndex(key)
					elem = derefed.MapIndex(key)
					found = true
					break
				}
			}
			if !found {
				return elem, sel, fmt.Errorf("invalid path: %s", path)
			}
		default:
			return elem, sel, fmt.Errorf("unrecognized type: %s", derefed.Kind())
		}

		if elem.Kind() == reflect.Invalid {
			return elem, sel, fmt.Errorf("invalid path: %s", path)
		}
		components = components[1:]
	}

	return elem, components[0], nil
}
