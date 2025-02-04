package template

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	tests := map[string]struct {
		str         string
		expectError bool
	}{
		"success": {
			str: `{{a.b[0]}}`,
		},
		"failed": {
			str:         `{{}`,
			expectError: true,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			_, err := New(test.str)
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected error but got no error")
			}
		})
	}
}

func TestTemplate_Execute(t *testing.T) {
	tests := map[string]struct {
		str         string
		data        interface{}
		expect      interface{}
		expectError bool
	}{
		"no parameter": {
			str:    "1",
			expect: "1",
		},
		"empty parameter": {
			str:    "{{}}",
			expect: "",
		},
		"empty parameter with string": {
			str:    "prefix-{{}}-suffix",
			expect: "prefix--suffix",
		},
		"string": {
			str:    `{{"foo"}}`,
			expect: "foo",
		},
		"integer": {
			str:    "{{1}}",
			expect: 1,
		},
		"add": {
			str:    `foo-{{ "bar" + "-" + "baz" }}`,
			expect: "foo-bar-baz",
		},
		"query from data": {
			str: "{{a.b[1]}}",
			data: map[string]map[string][]string{
				"a": map[string][]string{
					"b": []string{"ng", "ok"},
				},
			},
			expect: "ok",
		},
		"function call": {
			str: `{{f("ok")}}`,
			data: map[string]func(string) string{
				"f": func(s string) string { return s }},
			expect: "ok",
		},
		"not found": {
			str:         "{{a.b[1]}}",
			expectError: true,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			tmpl, err := New(test.str)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			i, err := tmpl.Execute(test.data)
			if !test.expectError && err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if test.expectError && err == nil {
				t.Fatal("expected error but got no error")
			}
			if diff := cmp.Diff(test.expect, i); diff != "" {
				t.Errorf("diff: (-want +got)\n%s", diff)
			}
		})
	}
}
