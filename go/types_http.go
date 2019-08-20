package main

/*
TODO consider moving this to a library, along with the `withRes` family of
functions. (Might also redesign the API.)
*/

import (
	"net/http"
	"reflect"

	"github.com/mitranim/refut"
	"github.com/mitranim/reqdec"
	"github.com/mitranim/try"
	"github.com/pkg/errors"
)

// "Request decoder". Wrapper with better errors and some additional methods.
type Reqdec struct{ reqdec.Reqdec }

func ReqdecFromReqQuery(req *Req) Reqdec {
	return Reqdec{reqdec.FromReqQuery(req)}
}

func DownloadReqdec(req *Req) (Reqdec, error) {
	dec, err := reqdec.Download(req)
	return Reqdec{dec}, ErrPubBadRequest(errors.WithStack(err))
}

func TryDownloadReqdec(req *Req) Reqdec {
	dec, err := DownloadReqdec(req)
	try.To(err)
	return dec
}

func (self Reqdec) DecodeStruct(dest interface{}) error {
	err := self.Reqdec.DecodeStruct(dest)
	return ErrPubBadRequest(errors.WithStack(err))
}

func (self Reqdec) DecodeAt(key string, dest interface{}) error {
	err := self.Reqdec.DecodeAt(key, dest)
	return ErrPubBadRequest(errors.WithStack(err))
}

func (self Reqdec) DecodeValidateStruct(out Validator) error {
	err := self.DecodeStruct(out)
	if err != nil {
		return err
	}
	return ErrPubBadRequest(out.Validate())
}

/*
Variant of `sqlb.StructNamedArgs` that returns only the fields present in the
request.
*/
func (self Reqdec) StructSqlArgs(input interface{}) Args {
	var args Args

	try.To(refut.TraverseStruct(input, func(rval reflect.Value, sfield reflect.StructField, _ []int) error {
		if !self.Has(sfieldJsonFieldName(sfield)) {
			return nil
		}
		args = append(args, Arg(sfieldDbColName(sfield), rval.Interface()))
		return nil
	}))

	return args
}

type HttpReqParams struct {
	Method string
	Url    string
	Header http.Header
	Body   []byte
}

type JsonReqParams struct {
	Method string
	Url    string
	Header http.Header
	Body   interface{}
}

func (self JsonReqParams) HttpReqParams() (HttpReqParams, error) {
	body, err := maybeJsonMarshal(self.Body)
	if err != nil {
		return HttpReqParams{}, err
	}

	return HttpReqParams{
		Method: self.Method,
		Url:    self.Url,
		Header: patchHttpHeader(jsonResHeader, self.Header),
		Body:   body,
	}, nil
}
