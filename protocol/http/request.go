package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/context"
	"github.com/zoncoen/scenarigo/internal/reflectutil"
	"github.com/zoncoen/scenarigo/protocol/http/marshaler"
	"github.com/zoncoen/scenarigo/protocol/http/unmarshaler"
)

// Request represents a request.
type Request struct {
	Client string      `yaml:"client"`
	Method string      `yaml:"method"`
	URL    string      `yaml:"url"`
	Header interface{} `yaml:"header"`
	Body   interface{} `yaml:"body"`
}

// Invoke implements protocol.Invoker interface.
func (r *Request) Invoke(ctx *context.Context) (*context.Context, interface{}, error) {
	client, err := r.buildClient(ctx)
	if err != nil {
		return ctx, nil, err
	}
	req, reqBody, err := r.buildRequest(ctx)
	if err != nil {
		return ctx, nil, err
	}
	ctx = ctx.WithRequest(reqBody)

	resp, err := client.Do(req)
	if err != nil {
		return ctx, nil, errors.Errorf("failed to send request: %s", err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ctx, nil, errors.Errorf("failed to read response body: %s", err)
	}

	var respBody interface{}
	if len(b) > 0 {
		unmarshaler := unmarshaler.Get(resp.Header.Get("Content-Type"))
		if err := unmarshaler.Unmarshal(b, &respBody); err != nil {
			return ctx, nil, errors.Errorf("failed to unmarshal response body as %s: %s: %s", unmarshaler.MediaType(), string(b), err)
		}
		ctx = ctx.WithResponse(respBody)
	}

	return ctx, newResult(resp, respBody), nil
}

func (r *Request) buildClient(ctx *context.Context) (*http.Client, error) {
	client := &http.Client{}
	if r.Client != "" {
		x, err := ctx.ExecuteTemplate(r.Client)
		if err != nil {
			return nil, errors.Errorf("failed to get client: %s", err)
		}
		var ok bool
		if client, ok = x.(*http.Client); !ok {
			return nil, errors.Errorf(`client must be "*http.Client" but got "%T"`, x)
		}
	}
	return client, nil
}

func (r *Request) buildRequest(ctx *context.Context) (*http.Request, interface{}, error) {
	method := http.MethodGet
	if r.Method != "" {
		method = r.Method
	}

	x, err := ctx.ExecuteTemplate(r.URL)
	if err != nil {
		return nil, nil, errors.Errorf("failed to get URL: %s", err)
	}
	url, ok := x.(string)
	if !ok {
		return nil, nil, errors.Errorf(`URL must be "string" but got "%T"`, x)
	}

	header := http.Header{}
	if r.Header != nil {
		x, err := ctx.ExecuteTemplate(r.Header)
		if err != nil {
			return nil, nil, errors.Errorf("failed to set header: %s", err)
		}
		hdr, err := reflectutil.ConvertStringsMap(reflect.ValueOf(x))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to set header")
		}
		for k, vs := range hdr {
			vs := vs
			for _, v := range vs {
				header.Add(k, v)
			}
		}
	}

	var reader io.Reader
	var body interface{}
	if r.Body != nil {
		x, err := ctx.ExecuteTemplate(r.Body)
		if err != nil {
			return nil, nil, errors.Errorf("failed to create request: %s", err)
		}
		body = x

		marshaler := marshaler.Get(header.Get("Content-Type"))
		b, err := marshaler.Marshal(body)
		if err != nil {
			return nil, nil, errors.Errorf("failed to marshal request body as %s: %#v: %s", marshaler.MediaType(), body, err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, nil, errors.Errorf("failed to create request: %s", err)
	}
	req = req.WithContext(ctx.RequestContext())

	for k, vs := range header {
		vs := vs
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	return req, body, nil
}
