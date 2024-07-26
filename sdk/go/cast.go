package sdk

import (
	"fmt"

	"github.com/spf13/cast"
)

func (r Object) GetString(keys ...string) (string, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return "", err
	}
	if len(keys) == 1 {
		return cast.ToStringE(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return "", err
	}

	return r1.GetString(keys[1:]...)
}

func (r Object) GetInt(keys ...string) (int, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return 0, err
	}
	if len(keys) == 1 {
		return cast.ToIntE(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return 0, err
	}

	return r1.GetInt(keys[1:]...)
}

func (r Object) GetBool(keys ...string) (bool, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return false, err
	}
	if len(keys) == 1 {
		return cast.ToBoolE(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return false, err
	}

	return r1.GetBool(keys[1:]...)
}

func (r Object) GetFloat64(keys ...string) (float64, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return 0, err
	}
	if len(keys) == 1 {
		return cast.ToFloat64E(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return 0, err
	}

	return r1.GetFloat64(keys[1:]...)
}

func (r Object) GetInt64(keys ...string) (int64, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return 0, err
	}
	if len(keys) == 1 {
		return cast.ToInt64E(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return 0, err
	}

	return r1.GetInt64(keys[1:]...)
}

func (r Object) GetSlice(keys ...string) ([]interface{}, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return nil, err
	}
	if len(keys) == 1 {
		return cast.ToSliceE(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return nil, err
	}

	return r1.GetSlice(keys[1:]...)
}

func (r Object) GetStringMap(keys ...string) (Object, error) {

	err := r.keyCheck(keys...)
	if err != nil {
		return nil, err
	}
	if len(keys) == 1 {
		return cast.ToStringMapE(r[keys[0]])
	}

	var r1 Object
	r1, err = cast.ToStringMapE(r[keys[0]])
	if err != nil {
		return nil, err
	}

	return r1.GetStringMap(keys[1:]...)
}

func (r Object) keyCheck(keys ...string) error {
	if len(keys) == 0 {
		return fmt.Errorf("no keys provided")
	}
	if _, ok := r[keys[0]]; !ok {
		return fmt.Errorf("key %s not found", keys[0])
	}
	return nil
}
