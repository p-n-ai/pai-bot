// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package jsonobject provides a type-safe representation of JSON objects with
// heterogeneous members. Values are encoded when added and decoded into an
// explicit caller-owned type when read.
package jsonobject

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Field struct {
	name  string
	value json.RawMessage
	err   error
}

type Object struct {
	fields map[string]json.RawMessage
	err    error
}

func Member[T any](name string, value T) Field {
	encoded, err := json.Marshal(value)
	if err != nil {
		return Field{name: name, err: fmt.Errorf("encode JSON member %q: %w", name, err)}
	}
	return Field{name: name, value: encoded}
}

func New(fields ...Field) Object {
	object := Object{fields: make(map[string]json.RawMessage, len(fields))}
	for _, field := range fields {
		if object.err == nil && field.err != nil {
			object.err = field.err
		}
		if field.name != "" {
			object.fields[field.name] = append(json.RawMessage(nil), field.value...)
		}
	}
	return object
}

func From[T any](value T) Object {
	encoded, err := json.Marshal(value)
	if err != nil {
		return Object{err: fmt.Errorf("encode JSON object: %w", err)}
	}
	parsed, err := Parse(encoded)
	if err != nil {
		return Object{err: err}
	}
	return parsed
}

func Parse(encoded []byte) (Object, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return Object{}, fmt.Errorf("decode JSON object: %w", err)
	}
	if fields == nil {
		return Object{}, fmt.Errorf("JSON value must be an object")
	}
	return Object{fields: fields}, nil
}

func Get[T any](object Object, name string) (T, bool, error) {
	var value T
	encoded, ok := object.fields[name]
	if !ok {
		return value, false, nil
	}
	if err := json.Unmarshal(encoded, &value); err != nil {
		return value, true, fmt.Errorf("decode JSON member %q: %w", name, err)
	}
	return value, true, nil
}

func (o Object) With(field Field) Object {
	fields := make(map[string]json.RawMessage, len(o.fields)+1)
	for name, value := range o.fields {
		fields[name] = append(json.RawMessage(nil), value...)
	}
	updated := Object{fields: fields, err: o.err}
	if updated.err == nil && field.err != nil {
		updated.err = field.err
	}
	if field.name != "" {
		updated.fields[field.name] = append(json.RawMessage(nil), field.value...)
	}
	return updated
}

func (o Object) Len() int {
	return len(o.fields)
}

func (o Object) MarshalJSON() ([]byte, error) {
	if o.err != nil {
		return nil, o.err
	}
	if o.fields == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(o.fields)
}

func (o *Object) UnmarshalJSON(encoded []byte) error {
	parsed, err := Parse(encoded)
	if err != nil {
		return err
	}
	*o = parsed
	return nil
}

func (o Object) Equal(other Object) bool {
	left, leftErr := o.MarshalJSON()
	right, rightErr := other.MarshalJSON()
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}
