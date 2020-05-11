package utils

import (
	"reflect"
	"strings"
)

func SetByPath(obj interface{}, prop string, value interface{}) error {
	arr := strings.Split(prop, ".")
	var err error
	var key string
	last, arr := arr[len(arr)-1], arr[:len(arr)-1]
	for _, key = range arr {
		obj, err = search(obj, key)
		if err != nil {
			return err
		}
	}

	if obj == nil {
		return nil
	}

	if reflect.TypeOf(obj).Kind() == reflect.Map {
		src := reflect.ValueOf(obj)
		src.SetMapIndex(reflect.ValueOf(last), reflect.ValueOf(value))
	}

	return nil
}

func search(obj interface{}, prop string) (interface{}, error) {
	val := reflect.ValueOf(obj)
	if !val.IsValid() {
		return nil, nil
	}

	valueOf := val.MapIndex(reflect.ValueOf(prop))
	if valueOf == reflect.Zero(reflect.ValueOf(prop).Type()) {
		return nil, nil
	}

	idx := val.MapIndex(reflect.ValueOf(prop))
	if !idx.IsValid() {
		return nil, nil
	}

	return idx.Interface(), nil
}
