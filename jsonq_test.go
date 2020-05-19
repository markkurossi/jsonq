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
        "changelog": {
            "items": [
                {
                    "fieldId": "assignee",
                    "toString": "Veijo Linux",
                    "fromString": null
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
	val, err = GetString(v, "issue.key")
	if err != nil {
		t.Fatalf("string value not found: %s", err)
	}
	if val != "OP-1" {
		t.Fatalf("invalid string value: got %s, expected %s", val, "OP-1")
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
