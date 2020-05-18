//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"fmt"
	"strings"
)

// GetString gets the string value pointed by the query q.
func GetString(value interface{}, q string) (string, error) {
	var optional bool
	if strings.HasPrefix(q, "?") {
		optional = true
		q = q[1:]
	}
	v, err := GetOne(value, q)
	if err != nil {
		if optional {
			return "", nil
		}
		return "", err
	}
	switch val := v.(type) {
	case string:
		return val, nil

	case nil:
		if optional {
			return "", nil
		}
	}

	return "", fmt.Errorf("jsonq: value of '%s' is not string: %T", q, v)
}

func GetOne(value interface{}, q string) (interface{}, error) {
	result, err := Get(value, q)
	if err != nil {
		return nil, err
	}
	switch len(result) {
	case 0:
		return nil, fmt.Errorf("jsonq: element '%s' not found", q)

	case 1:
		return result[0], nil

	default:
		return nil, fmt.Errorf("json: multiple results for '%s'", q)
	}
}

func Get(value interface{}, q string) ([]interface{}, error) {
	parts := strings.Split(q, ".")
	for idx, part := range parts {
		var key, filters string
		ofs := strings.IndexByte(part, '[')
		if ofs >= 0 {
			key = part[:ofs]
			filters = part[ofs:]
		} else {
			key = part
		}

		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("jsonq: query '%s' can't index %T",
				strings.Join(append(parts[:idx], key), "."), value)
		}

		child, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("jsonq: element '%s' not found",
				strings.Join(append(parts[:idx], key), "."))
		}
		value = child
		if len(filters) > 0 {
			return filter(child, filters)
		}
	}
	return []interface{}{value}, nil
}

func filter(value interface{}, filter string) ([]interface{}, error) {
	filter = strings.TrimPrefix(filter, "[")
	filter = strings.TrimSuffix(filter, "]")

	filters := strings.Split(filter, "][")

	var result []interface{}
	arr, ok := value.([]interface{})
	if ok {
		result = arr
	} else {
		result = []interface{}{value}
	}

	for _, f := range filters {
		q, err := Parse(f)
		if err != nil {
			return nil, err
		}
		next, err := q.Eval(result)
		if err != nil {
			return nil, err
		}
		result = next
	}
	return result, nil
}
