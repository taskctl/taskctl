package util

import (
	"errors"
	"gopkg.in/oleiade/reflections.v1"
	"reflect"
	"strings"
)

func GetByPath(key string, src interface{}) (value interface{}, err error) {
	path := strings.Split(key, ".")

	for i := 0; i < len(path); i++ {
		src, err = getField(path[i], src)
		if err != nil {
			return nil, err
		}
	}

	return src, nil
}

func SetByPath(key string, value interface{}, src interface{}) (err error) {
	defer (func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	})()

	path := strings.Split(key, ".")

	src, err = GetByPath(strings.Join(path[:len(path)-1], "."), src)
	if err != nil {
		return err
	}

	if src == nil {
		return errors.New("key not found or not initialized")
	}

	key = path[len(path)-1]
	if reflect.TypeOf(src).Kind() == reflect.Map {
		reflect.ValueOf(src).SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		return nil
	}

	if reflect.TypeOf(src).Kind() != reflect.Ptr {
		return errors.New("object must be a pointer to a struct")
	}
	key = strings.Title(key)

	return reflections.SetField(src, key, value)
}

func getField(key string, src interface{}) (value interface{}, err error) {
	if reflect.TypeOf(src).Kind() == reflect.Map {

		val := reflect.ValueOf(src)

		valueOf := val.MapIndex(reflect.ValueOf(key))

		if valueOf == reflect.Zero(reflect.ValueOf(key).Type()) {
			return nil, nil
		}

		idx := val.MapIndex(reflect.ValueOf(key))

		if !idx.IsValid() {
			return nil, nil
		}
		return idx.Interface(), nil
	}

	key = strings.Title(key)
	return reflections.GetField(src, key)
}
