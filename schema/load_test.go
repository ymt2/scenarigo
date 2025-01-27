package schema

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zoncoen/scenarigo/protocol"
)

type testProtocol struct {
	name            string
	request, expect interface{}
}

func (p *testProtocol) Name() string { return p.name }

func (p *testProtocol) UnmarshalRequest(f func(interface{}) error) (protocol.Invoker, error) {
	if err := f(&p.request); err != nil {
		return nil, err
	}
	return nil, nil
}

func (p *testProtocol) UnmarshalExpect(f func(interface{}) error) (protocol.AssertionBuilder, error) {
	if err := f(&p.expect); err != nil {
		return nil, err
	}
	return nil, nil
}

func TestLoadScenarios(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			path            string
			scenarios       []*Scenario
			request, expect interface{}
		}{
			"valid": {
				path: "testdata/valid.yaml",

				scenarios: []*Scenario{
					{
						Title:       "echo-service",
						Description: "check echo-service",
						Vars:        map[string]interface{}{"message": "hello"},
						Steps: []*Step{
							{
								Title:       "POST /say",
								Description: "check to respond same message",
								Vars:        nil,
								Protocol:    "test",
							},
						},
						filepath: "testdata/valid.yaml",
					},
				},
				request: map[interface{}]interface{}{
					"body": map[interface{}]interface{}{"message": "{{vars.message}}"},
				},
				expect: map[interface{}]interface{}{
					"body": map[interface{}]interface{}{"message": "{{request.body}}"},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				p := &testProtocol{
					name:    "test",
					request: map[interface{}]interface{}{},
					expect:  map[interface{}]interface{}{},
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				got, err := LoadScenarios(test.path)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if diff := cmp.Diff(test.scenarios, got,
					cmp.AllowUnexported(
						Scenario{}, Request{}, Expect{},
					),
					cmp.FilterPath(func(path cmp.Path) bool {
						s := path.String()
						if s == "Steps.Request" {
							return true
						}
						if s == "Steps.Expect" {
							return true
						}
						return false
					}, cmp.Ignore()),
				); diff != "" {
					t.Errorf("scenario differs (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.request, p.request); diff != "" {
					t.Errorf("request differs (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(test.expect, p.expect); diff != "" {
					t.Errorf("expect differs (-want +got):\n%s", diff)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		p := &testProtocol{
			name: "test",
		}
		protocol.Register(p)
		defer protocol.Unregister(p.Name())

		tests := map[string]struct {
			path string
		}{
			"not found": {
				path: "notfound.yaml",
			},
			"invalid": {
				path: "testdata/invalid.yaml",
			},
			"unknown protocol": {
				path: "testdata/unknown-protocol.yaml",
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				_, err := LoadScenarios(test.path)
				if err == nil {
					t.Fatal("expected error but no error")
				}
			})
		}
	})
}
