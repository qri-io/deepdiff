package deepdiff

import (
	"fmt"
	"reflect"
	"strconv"
)

// Patch applies a change script (patch) to a value
func Patch(deltas []*Delta, target interface{}) error {
	t := reflect.ValueOf(target)
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("must pass a pointer value to patch")
	}
	t = t.Elem()

	for _, dlt := range deltas {
		patched, err := patch(t, dlt)
		if err != nil {
			return err
		}
		t.Set(patched)
	}

	return nil
}

func patch(target reflect.Value, delta *Delta) (reflect.Value, error) {
	var err error
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	// traverse the tree bottom-up, setting each parent
	// value to the updated child
	if len(delta.Deltas) > 0 {
		for _, dlt := range delta.Deltas {
			// fmt.Printf("patching %s on target %#v\n", dlt.Path, target)
			patchedChild, err := patch(child(target, delta.Path), dlt)
			if err != nil {
				return target, err
			}

			// fmt.Printf("patch output: %#v\n", patchedChild)

			target, err = set(target, patchedChild, delta.Path)
			if err != nil {
				return target, err
			}

			// fmt.Printf("post-patch-set target: %#v\n\n", target)
		}
	}

	switch delta.Type {
	case DTInsert:
		// fmt.Printf("applying insert to %s on target %#v\n", delta.Path, target)
		target, err = insert(target, reflect.ValueOf(delta.Value), delta.Path)
	case DTDelete:
		// fmt.Printf("applying delete to %s on target %#v\n", delta.Path, target)
		target, err = remove(target, delta.Path)
		// fmt.Printf("post-delete target %#v\n", target)
	case DTUpdate:
		// fmt.Printf("applying update to %s on target %#v\n", delta.Path, target)
		target, err = set(target, reflect.ValueOf(delta.Value), delta.Path)
	}

	return target, err
}

func set(target, value reflect.Value, key string) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		target.SetMapIndex(reflect.ValueOf(key), value)
	case reflect.Slice:
		i, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		l := target.Len()
		sl := reflect.MakeSlice(target.Type(), 0, l)
		sl = reflect.AppendSlice(sl, target.Slice(0, i))
		sl = reflect.Append(sl, value)
		if i < l {
			sl = reflect.AppendSlice(sl, target.Slice(i+1, l))
		}

		target = sl
	}

	return target, nil
}

func remove(target reflect.Value, key string) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		// SetMapIndex expects a zero value for reflect.Value itself to delete a key
		target.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
	case reflect.Slice:
		i, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		l := target.Len()
		sl := reflect.MakeSlice(target.Type(), 0, l)
		sl = reflect.AppendSlice(sl, target.Slice(0, i))
		sl = reflect.AppendSlice(sl, target.Slice(i+1, l))

		target = sl
	}

	return target, nil
}

func insert(target, value reflect.Value, key string) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		target.SetMapIndex(reflect.ValueOf(key), value)
	case reflect.Slice:
		i, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		l := target.Len()
		sl := reflect.MakeSlice(target.Type(), 0, l)
		sl = reflect.AppendSlice(sl, target.Slice(0, i))
		sl = reflect.Append(sl, value)
		sl = reflect.AppendSlice(sl, target.Slice(i, l))

		target = sl
	}

	return target, nil
}

func child(target reflect.Value, key string) reflect.Value {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		target = target.MapIndex(reflect.ValueOf(key))
	case reflect.Slice:
		i, err := strconv.Atoi(key)
		if err != nil {
			panic(err)
		}
		target = target.Index(i)
	}

	return target
}

func descendant(target reflect.Value, keys []string) reflect.Value {
	if len(keys) == 0 {
		return target
	}

	for _, key := range keys {
		if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
			target = target.Elem()
		}

		switch target.Kind() {
		case reflect.Map:
			target = target.MapIndex(reflect.ValueOf(key))
		case reflect.Slice:
			i, err := strconv.Atoi(key)
			if err != nil {
				panic(err)
			}
			target = target.Index(i)
		}
	}
	return target
}

// // Patch applies a change script (patch) to a value
// func Patch(v interface{}, patch []*Delta) (err error) {
// 	rv := reflect.ValueOf(v)
// 	if rv.Kind() != reflect.Ptr || rv.IsNil() {
// 		return fmt.Errorf("passed in value must be a pointer")
// 	}

// 	for i, dlt := range patch {
// 		if err := applyDelta(rv.Elem(), dlt); err != nil {
// 			return fmt.Errorf("patch %d: %s", i, err)
// 		}
// 	}
// 	return nil
// }

// func applyDelta(tree reflect.Value, dlt *Delta) error {
// 	fmt.Printf("delta : %#v\nparent: %#v\n", dlt, tree)

// 	if len(dlt.Deltas) > 0 {
// 		parent, err := childValue(tree, dlt.Path)
// 		if err != nil {
// 			return err
// 		}

// 		for _, d := range dlt.Deltas {
// 			if err = applyDelta(parent, d); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	switch dlt.Type {
// 	case DTContext:
// 		return nil
// 	case DTUpdate:
// 		return updateValue(tree, dlt.Path, dlt.Value)
// 	case DTInsert:
// 		return insertValue(tree, dlt.Path, dlt.Value)
// 	case DTDelete:
// 		return deleteValue(tree, dlt.Path)
// 	case DTMove:
// 		return moveValue(tree, dlt.SourcePath, dlt.Path, dlt.Value)
// 	default:
// 		return fmt.Errorf("unknown delta type")
// 	}
// }

// func updateValue(val reflect.Value, key string, set interface{}) error {

// 	if val.Kind() == reflect.Interface {
// 		val = val.Elem()
// 	}

// 	switch val.Kind() {
// 	case reflect.Map:
// 		val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(set))
// 	case reflect.Slice:
// 		i, err := strconv.Atoi(key)
// 		if err != nil {
// 			return err
// 		}
// 		val.Index(i).Set(reflect.ValueOf(set))
// 	default:
// 		return fmt.Errorf("unrecognized data type for path '%s': %T", key, val)
// 	}

// 	return nil
// }

// func deleteValue(val reflect.Value, key string) error {
// 	if val.Kind() == reflect.Interface {
// 		val = val.Elem()
// 	}

// 	fmt.Printf("\tdelete value\n\t\tvalue: %#v\n\t\tkey: %s\n", val, key)

// 	switch val.Kind() {
// 	case reflect.Map:
// 		val.SetMapIndex(reflect.ValueOf(path), reflect.Value{})
// 	case reflect.Slice:
// 		i, err := strconv.Atoi(key)
// 		if err != nil {
// 			return err
// 		}

// 		l := val.Len()
// 		if i >= l {
// 			return fmt.Errorf("array index %d exceeds %d at path %s", i, l, key)
// 		} else if i == l-1 {
// 			val.Set(val.Slice(0, l-1))
// 		} else {
// 			val.Set(reflect.AppendSlice(val.Slice(0, i), val.Slice(i+1, l)))
// 		}
// 	default:
// 		return fmt.Errorf("unrecognized data type for path '%s': %T", key, val)
// 	}

// 	return nil
// }

// func insertValue(val reflect.Value, key string, insert interface{}) error {
// 	if val.Kind() == reflect.Interface {
// 		val = val.Elem()
// 	}

// 	// fmt.Printf("insert value: %#v\ntree: %#v\npath: %s\npare: %#v\nname: %s\nval:  %#v\n\n", insert, tree, path, parent, name, val)

// 	switch val.Kind() {
// 	case reflect.Map:
// 		val.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(insert))
// 	case reflect.Slice:
// 		i, err := strconv.Atoi(key)
// 		if err != nil {
// 			return err
// 		}
// 		l := val.Len()
// 		if i > l {
// 			return fmt.Errorf("array index %d exceeds %d at path %s", i, l, key)
// 		} else if i == l {
// 			val.Set(reflect.Append(val, reflect.ValueOf(insert)))
// 		} else {
// 			// TODO (b5): reflect.Append seems to make descructive edits to val if used directly
// 			// so we need to copy before append. really need to find a concise way to say the following
// 			// using reflection:
// 			// parent = append(append(val[i:], elem), val[i:]...)
// 			slcp := reflect.MakeSlice(val.Type(), i, i)
// 			reflect.Copy(slcp, val.Slice(0, i))
// 			val.Set(
// 				reflect.AppendSlice(
// 					reflect.Append(slcp, reflect.ValueOf(insert)),
// 					val.Slice(i, l),
// 				),
// 			)
// 		}
// 	default:
// 		return fmt.Errorf("unrecognized data type for path '%s': %T", key, val)
// 	}

// 	return nil
// }

// func moveValue(tree reflect.Value, from, to string, val interface{}) error {
// 	if err := deleteValue(tree, from); err != nil {
// 		return err
// 	}
// 	return insertValue(tree, to, val)
// }

// func childValue(val reflect.Value, path string) (reflect.Value, error) {

// 	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
// 		val = val.Elem()
// 	}

// 	switch val.Kind() {
// 	case reflect.Slice:
// 		index, err := strconv.Atoi(path)
// 		if err != nil {
// 			return val, fmt.Errorf("invalid index value: %s", path)
// 		}
// 		val = val.Index(index)
// 	case reflect.Map:
// 		found := false
// 		for _, key := range val.MapKeys() {
// 			if key.String() == path {
// 				val = val.MapIndex(key)
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			return val, fmt.Errorf("invalid path: %s", path)
// 		}
// 	default:
// 		return val, fmt.Errorf("unrecognized type: %s", val.Kind())
// 	}

// 	if val.Kind() == reflect.Invalid {
// 		return val, fmt.Errorf("invalid path: %s", path)
// 	}

// 	return val, nil
// }
