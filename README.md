# jsonq

JSONQ is XPath inspired JSON processing library for Go. It provides
functions for traversing JSON structures and for extracting attributes
into Go data structures.

[![Build Status](https://img.shields.io/github/workflow/status/markkurossi/jsonq/Go)](https://github.com/markkurossi/jsonq/actions)
[![Git Hub](https://img.shields.io/github/last-commit/markkurossi/jsonq.svg)](https://github.com/markkurossi/jsonq/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/markkurossi/jsonq)](https://goreportcard.com/report/github.com/markkurossi/jsonq)

## Type-safe getters

Type-safe getters access JSON structures and return type-checked values.

```go
var v interface{}
err := json.Unmarshal([]byte(`{
    "issue": {
        "fields": {
            "project": {
                "name": "Operations"
            }
        }
    }
}`), &v)
if err != nil {
    log.Fatal(err)
}
name, err := GetString(v, "issue.fields.project.name")
if err != nil {
    log.Fatal(err)
}
fmt.Println(name)
// Output: Operations
```

## Extracting JSON attributes to Go data structures

The Context type allows you to select elements from JSON data and
extract values into Go data structures. The basic extraction works by
annotating Go structurs with the `jsonq` structure tags:

```go
var v interface{}
err := json.Unmarshal([]byte(`{
    "issue_event_type_name": "issue_assigned",
    "issue": {
        "fields": {
            "project": {
                "name": "Operations"
            }
        },
        "key": "OP-1"
    }
}`), &v)
if err != nil {
    log.Fatal(err)
}
var issue struct {
    Key       string `jsonq:"issue.key"`
    Name      string `jsonq:"issue.fields.project.name"`
    EventType string `jsonq:"issue_event_type_name"`
}
err = Ctx(v).Extract(&issue)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("key=%s, name=%s, type=%s\n", issue.Key, issue.Name, issue.EventType)
// Output: key=OP-1, name=Operations, type=issue_assigned
```

The Context.Select() function allows you to filter data based on JSON
attributes:

```go
var v interface{}
err := json.Unmarshal([]byte(`{
    "issue": {
        "changelog": {
            "items": [
                {
                    "fieldId": "comment",
                    "toString": "This is cool!",
                    "fromString": ""
                },
                {
                    "fieldId": "assignee",
                    "toString": "Veijo Linux",
                    "fromString": null
                }
            ]
        }
    }
}`), &v)
if err != nil {
    log.Fatal(err)
}
var assign struct {
    From string `jsonq:"?fromString"`
    To   string `jsonq:"?toString"`
}
err = Ctx(v).
    Select(`issue.changelog.items[fieldId=="assignee"][0]`).
    Extract(&assign)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("assign from '%s' to '%s'\n", assign.From, assign.To)
// Output: assign from '' to 'Veijo Linux'
```

Note that if the JSON attribute name is prefixed with question mark,
the field is optional.

## TODO

 - Getters:
   - [X] Number
   - [X] Boolean
 - Expressions:
   - [X] Not equal
   - [ ] Number comparison: ==, !=, <, > <=, >=
   - [ ] Parenthesized sub-expressions
   - [ ] Chaining logical expressions
