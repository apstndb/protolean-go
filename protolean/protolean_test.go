package protolean

import (
	"strings"
	"testing"

	"github.com/apstndb/protolean-go/testdata"
)

func TestMarshalPerson(t *testing.T) {
	p := &testdata.Person{
		Name:   "Alice",
		Age:    30,
		Active: true,
		Tags:   []string{"go", "proto"},
		Status: testdata.Status_STATUS_ACTIVE,
	}

	got, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "name:Alice") {
		t.Errorf("expected name field")
	}
	if !strings.Contains(got, "age:30") {
		t.Errorf("expected age field")
	}
	if !strings.Contains(got, "active:T") {
		t.Errorf("expected active as T")
	}
	if !strings.Contains(got, "tags[2]:") {
		t.Errorf("expected tags array")
	}
}

func TestMarshalCompany(t *testing.T) {
	c := &testdata.Company{
		Name: "Acme",
		Employees: []*testdata.Person{
			{Name: "Alice", Age: 30, Active: true, Status: testdata.Status_STATUS_ACTIVE},
			{Name: "Bob", Age: 25, Active: false, Status: testdata.Status_STATUS_INACTIVE},
		},
		Metadata: map[string]string{
			"industry": "tech",
			"size":     "small",
		},
	}

	got, err := Marshal(c)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("LEAN:\n%s", got)

	if !strings.Contains(got, "name:Acme") {
		t.Errorf("expected company name")
	}
	if !strings.Contains(got, "employees[2]:") {
		t.Errorf("expected employees array")
	}
	if !strings.Contains(got, "metadata:") {
		t.Errorf("expected metadata")
	}
}

func TestMarshalAllTypes(t *testing.T) {
	m := &testdata.AllTypes{
		DoubleField:   1.5,
		FloatField:    2.5,
		Int32Field:    -10,
		Int64Field:    -20,
		Uint32Field:   30,
		Uint64Field:   40,
		Sint32Field:   -50,
		Sint64Field:   -60,
		Fixed32Field:  70,
		Fixed64Field:  80,
		Sfixed32Field: -90,
		Sfixed64Field: -100,
		BoolField:     true,
		StringField:   "hello",
		BytesField:    []byte("world"),
	}

	got, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	t.Logf("LEAN:\n%s", got)

	checks := []string{
		"double_field:1.5",
		"float_field:2.5",
		"int32_field:-10",
		"bool_field:T",
		"string_field:hello",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("expected %q in output", check)
		}
	}
}

func TestMarshalNil(t *testing.T) {
	got, err := Marshal(nil)
	if err != nil {
		t.Fatalf("Marshal(nil) failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
