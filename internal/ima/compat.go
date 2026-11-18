package ima

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Int64 accepts both JSON integers and quoted integers from IMA.
// It marshals back as a JSON number.
type Int64 int64

func (v *Int64) UnmarshalJSON(data []byte) error {
	n, err := parseLooseInt(data, 64)
	if err != nil {
		return err
	}
	*v = Int64(n)
	return nil
}

// Int accepts both JSON integers and quoted integers from IMA.
// It marshals back as a JSON number.
type Int int

func (v *Int) UnmarshalJSON(data []byte) error {
	n, err := parseLooseInt(data, strconv.IntSize)
	if err != nil {
		return err
	}
	*v = Int(n)
	return nil
}

// Bool accepts JSON booleans and quoted "true"/"false" values.
type Bool bool

func (v *Bool) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		*v = false
		return nil
	}
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*v = Bool(b)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("expected boolean or quoted boolean: %w", err)
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("invalid quoted boolean %q: %w", s, err)
	}
	*v = Bool(b)
	return nil
}

// String accepts JSON strings as well as scalar numbers/booleans.
// This is used for cursor/version fields that IMA may serialize inconsistently.
type String string

func (v *String) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		*v = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*v = String(s)
		return nil
	}
	var scalar any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&scalar); err != nil {
		return fmt.Errorf("expected string or scalar: %w", err)
	}
	switch x := scalar.(type) {
	case json.Number:
		*v = String(x.String())
	case bool:
		*v = String(strconv.FormatBool(x))
	default:
		return fmt.Errorf("expected string or scalar, got %T", scalar)
	}
	return nil
}

func parseLooseInt(data []byte, bits int) (int64, error) {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		return 0, nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err != nil {
		return 0, fmt.Errorf("expected integer or quoted integer: %w", err)
	}
	if n.String() == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(n.String(), 10, bits)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q: %w", n.String(), err)
	}
	return v, nil
}
