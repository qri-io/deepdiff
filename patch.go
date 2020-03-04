package deepdiff

import (
	"fmt"
	"reflect"
)

// Patch applies a change script (patch) to a value
func Patch(deltas Deltas, target interface{}) error {
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

func set(target, value reflect.Value, addr Addr) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		target.SetMapIndex(reflect.ValueOf(addr.Value()), value)
	case reflect.Slice:
		i, ok := addr.Value().(int)
		if !ok {
			panic("non-int value for slice address")
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

func remove(target reflect.Value, addr Addr) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	// root returns nil value, which is a total replace
	if addr.Value() == nil {
		// TODO (b5) - pass a real nil value here
		return reflect.ValueOf(map[string]interface{}{}), nil
	}

	switch target.Kind() {
	case reflect.Map:
		// SetMapIndex expects a zero value for reflect.Value itself to delete a key
		target.SetMapIndex(reflect.ValueOf(addr.Value()), reflect.Value{})
	case reflect.Slice:
		i, ok := addr.Value().(int)
		if !ok {
			panic("non-int value for slice address")
		}
		l := target.Len()
		sl := reflect.MakeSlice(target.Type(), 0, l)
		sl = reflect.AppendSlice(sl, target.Slice(0, i))
		sl = reflect.AppendSlice(sl, target.Slice(i+1, l))

		target = sl
	}

	return target, nil
}

func insert(target, value reflect.Value, addr Addr) (reflect.Value, error) {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	// root returns nil value, which is a total replace
	if addr.Value() == nil {
		return value, nil
	}

	switch target.Kind() {
	case reflect.Map:
		target.SetMapIndex(reflect.ValueOf(addr.Value()), value)
	case reflect.Slice:
		i, ok := addr.Value().(int)
		if !ok {
			panic("non-int value for slice address")
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

func child(target reflect.Value, addr Addr) reflect.Value {
	if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
		target = target.Elem()
	}

	switch target.Kind() {
	case reflect.Map:
		target = target.MapIndex(reflect.ValueOf(addr.Value()))
	case reflect.Slice:
		i, ok := addr.Value().(int)
		if !ok {
			panic("can't patch slice with non-int address value")
		}
		target = target.Index(i)
	}

	return target
}

func descendant(target reflect.Value, path []Addr) reflect.Value {
	if len(path) == 0 {
		return target
	}

	for _, addr := range path {
		if target.Kind() == reflect.Interface || target.Kind() == reflect.Ptr {
			target = target.Elem()
		}

		switch target.Kind() {
		case reflect.Map:
			target = target.MapIndex(reflect.ValueOf(addr.Value()))
		case reflect.Slice:
			i, ok := addr.Value().(int)
		if !ok {
			panic("can't patch slice with non-int address value")
		}
		target = target.Index(i)
		}
	}
	return target
}
