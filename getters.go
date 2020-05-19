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

func Get(value interface{}, q string) (interface{}, error) {
	query, err := Parse(q)
	if err != nil {
		return nil, err
	}
	return query.Eval(value)
}
