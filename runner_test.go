package scenarigo

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoncoen/scenarigo/assert"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/protocol"
	"github.com/zoncoen/scenarigo/reporter"
)

type testProtocol struct {
	name    string
	invoker invoker
	builder builder
}

func (p *testProtocol) Name() string { return p.name }

func (p *testProtocol) UnmarshalRequest(f func(interface{}) error) (protocol.Invoker, error) {
	return p.invoker, nil
}

func (p *testProtocol) UnmarshalExpect(f func(interface{}) error) (protocol.AssertionBuilder, error) {
	return p.builder, nil
}

type invoker func(*context.Context) (*context.Context, interface{}, error)

func (f invoker) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	return f(ctx)
}

type builder func(*context.Context) (assert.Assertion, error)

func (f builder) Build(ctx *context.Context) (assert.Assertion, error) {
	return f(ctx)
}

func TestRunner_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := map[string]struct {
			scenario string
			invoker  func(*context.Context) (*context.Context, interface{}, error)
			builder  func(*context.Context) (assert.Assertion, error)
		}{
			"simple": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder: func(ctx *context.Context) (assert.Assertion, error) {
					return assert.AssertionFunc(func(_ interface{}) error { return nil }), nil
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var invoked, built bool
				p := &testProtocol{
					name: "test",
					invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						invoked = true
						return test.invoker(ctx)
					}),
					builder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						built = true
						return test.builder(ctx)
					}),
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				r, err := NewRunner(WithScenarios(test.scenario))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				var b bytes.Buffer
				ok := reporter.Run(func(rptr reporter.Reporter) {
					r.Run(context.New(rptr))
				}, reporter.WithWriter(&b))
				if !ok {
					t.Fatalf("scenario failed:\n%s", b.String())
				}
				if !invoked {
					t.Error("did not invoke")
				}
				if !built {
					t.Error("did not build the assertion")
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			scenario string
			invoker  func(*context.Context) (*context.Context, interface{}, error)
			builder  func(*context.Context) (assert.Assertion, error)
		}{
			"failed to invoke": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker: func(ctx *context.Context) (*context.Context, interface{}, error) {
					return ctx, nil, errors.New("some error occurred")
				},
			},
			"failed to build the assertion": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder:  func(ctx *context.Context) (assert.Assertion, error) { return nil, errors.New("some error occurred") },
			},
			"assertion error": {
				scenario: "testdata/scenarios/simple.yaml",
				invoker:  func(ctx *context.Context) (*context.Context, interface{}, error) { return ctx, nil, nil },
				builder: func(ctx *context.Context) (assert.Assertion, error) {
					return assert.AssertionFunc(func(_ interface{}) error { return errors.New("some error occurred") }), nil
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				var invoked, built bool
				p := &testProtocol{
					name: "test",
					invoker: invoker(func(ctx *context.Context) (*context.Context, interface{}, error) {
						invoked = true
						return test.invoker(ctx)
					}),
					builder: builder(func(ctx *context.Context) (assert.Assertion, error) {
						built = true
						return test.builder(ctx)
					}),
				}
				protocol.Register(p)
				defer protocol.Unregister(p.Name())

				r, err := NewRunner(WithScenarios(test.scenario))
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				var b bytes.Buffer
				ok := reporter.Run(func(rptr reporter.Reporter) {
					r.Run(context.New(rptr))
				}, reporter.WithWriter(&b))
				if ok {
					t.Fatal("test passed")
				}
				if test.invoker != nil && !invoked {
					t.Error("did not invoke")
				}
				if test.builder != nil && !built {
					t.Error("did not build the assertion")
				}
			})
		}
	})
}