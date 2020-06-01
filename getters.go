//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"fmt"
)

// GetString gets the string value pointed by the query q.
func GetString(value interface{}, q string) (string, error) {
	v, err := Get(value, q)
	if err != nil {
		return "", err
	}
	switch val := v.(type) {
	case string:
		return val, nil

	case nil:
		return "", nil

	default:
		return "", fmt.Errorf("jsonq: value of '%s' is not string: %T", q, v)
	}
}

// GetNumber gets the float64 number value pointed by the query q.
func GetNumber(value interface{}, q string) (float64, error) {
	v, err := Get(value, q)
	if err != nil {
		return 0, err
	}
	switch val := v.(type) {
	case float64:
		return val, nil

	default:
		return 0, fmt.Errorf("jsonq: value of '%s' is not float64: %T", q, val)
	}
}

// GetInt gets the integer number value pointed by query q. The
// function internally gets the value as number and casts it to int
// type.
func GetInt(value interface{}, q string) (int, error) {
	v, err := GetNumber(value, q)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// GetBool gets the boolean value pointed by the query q.
func GetBool(value interface{}, q string) (bool, error) {
	v, err := Get(value, q)
	if err != nil {
		return false, err
	}
	switch val := v.(type) {
	case bool:
		return val, nil

	default:
		return false, fmt.Errorf("jsonq: value of '%s' is not bool: %T", q, val)
	}
}

// Get gets the values pointed by the query q.
func Get(value interface{}, q string) (interface{}, error) {
	query, err := parse(q)
	if err != nil {
		return nil, err
	}
	return query.Eval(value)
}
