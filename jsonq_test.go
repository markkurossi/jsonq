//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package jsonq

import (
	"encoding/json"
	"testing"
)

var assign = `{
    "issue_event_type_name": "issue_assigned",
    "issue": {
        "fields": {
            "project": {
                "name": "Operations"
            }
        },
        "key": "OP-1",
        "count": 42,
        "critical": false,
        "changelog": {
            "items": [
                {
                    "fieldId": "assignee",
                    "toString": "Veijo Linux",
                    "fromString": null
                },
                {
                    "fieldId": "assignee",
                    "toString": "Milton Waddams",
                    "fromString": "Veijo Linux"
                }
            ]
        }
    }
}
`

func TestGetters(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte(assign), &v)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %s", err)
	}

	_, err = Get(v, "nonexistent")
	if err == nil {
		t.Fatalf("non-existent element found")
	}
	val, err := GetString(v, "issue.count")
	if err == nil {
		t.Fatalf("integer value found as string: %s", val)
	}

	val, err = GetString(v, "issue.fields.project.name")
	if err != nil {
		t.Fatalf("string value not found: %s", err)
	}
	if val != "Operations" {
		t.Fatalf("invalid string value: got %s, expected %s", val, "Operations")
	}

	val, err = GetString(v, "issue.key")
	if err != nil {
		t.Fatalf("string value not found: %s", err)
	}
	if val != "OP-1" {
		t.Fatalf("invalid string value: got %s, expected %s", val, "OP-1")
	}

	nval, err := GetNumber(v, "issue.count")
	if err != nil {
		t.Fatalf("number value not found: %s", err)
	}
	if nval != float64(42) {
		t.Fatalf("invalid number value: got %v, expected %v", nval, float64(42))
	}

	ival, err := GetInt(v, "issue.count")
	if err != nil {
		t.Fatalf("int value not found: %s", err)
	}
	if ival != 42 {
		t.Fatalf("invalid int value: got %v, expected %v", ival, 42)
	}

	bval, err := GetBool(v, "issue.critical")
	if err != nil {
		t.Fatalf("boolean value not found: %s", err)
	}
	if bval != false {
		t.Fatalf("invalid boolean value: got %v, expected %v", ival, false)
	}
}

type Issue struct {
	Key       string `jsonq:"issue.key"`
	Name      string `jsonq:"issue.fields.project.name"`
	EventType string `jsonq:"issue_event_type_name"`
}

type Assignment struct {
	From string `jsonq:"?fromString"`
	To   string `jsonq:"?toString"`
	ID   string `jsonq:"?NonExistentID"`
}

func TestExtract(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte(assign), &v)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %s", err)
	}

	issue := new(Issue)
	err = Ctx(v).Extract(issue)
	if err != nil {
		t.Fatalf("Extract failed: %s", err)
	}
	if issue.Key != "OP-1" {
		t.Errorf("Invalid issue.key value")
	}
	if issue.Name != "Operations" {
		t.Errorf("Invalid issue.fields.project.name value")
	}
	if issue.EventType != "issue_assigned" {
		t.Errorf("Invalid issue_event_type_name value")
	}

	switch issue.EventType {
	case "issue_assigned":
		assignment := new(Assignment)
		err = Ctx(v).
			Select(`issue.changelog.items[fieldId=="assignee"][0]`).
			Extract(assignment)
		if err != nil {
			t.Fatalf("Extract failed: %s", err)
		}
		if assignment.From != "" {
			t.Errorf("Invalid value for assignment 'fromString' field")
		}
		if assignment.To != "Veijo Linux" {
			t.Errorf("Invalid value for assignment 'toString' field: %s",
				assignment.To)
		}
	}
}

func TestExtractArray(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte(assign), &v)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %s", err)
	}

	var history []Assignment
	err = Ctx(v).
		Select(`issue.changelog.items[fieldId=="assignee"]`).
		Extract(&history)
	if err != nil {
		t.Fatalf("Extract array failed: %s", err)
	}
	if len(history) != 2 {
		t.Fatalf("Extract array returned unexpected number of items: %v",
			history)
	}
	if history[0].To != "Veijo Linux" {
		t.Errorf("Invalid first array element: %v", history[0])
	}
	if history[1].To != "Milton Waddams" {
		t.Errorf("Invalid second array element: %v", history[1])
	}
}
