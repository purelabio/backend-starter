package main

import (
	"context"
	"reflect"
	"unsafe"

	"github.com/mitranim/refut"
	"github.com/mitranim/try"
	"github.com/pkg/errors"
)

/*
Allocation-free conversion. Reinterprets a byte slice as a string. Borrowed from
the standard library. Reasonably safe. Should not be used when the underlying
byte array is volatile, for example when it's part of a scratch buffer during
SQL scanning.
*/
func bytesToMutableString(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

// Self-reminder about non-free conversions.
func bytesToStringAlloc(bytes []byte) string { return string(bytes) }

// Self-reminder about non-free conversions.
func stringToBytesAlloc(input string) []byte { return []byte(input) }

func allowNotFound(err error) error {
	if isErrNotFound(err) {
		return nil
	}
	return err
}

func tryFound(err error) bool {
	if isErrNotFound(err) {
		return false
	}
	try.To(err)
	return true
}

// Note: this follows the conventional parameter order of "validate" functions
// where subject comes before predicates, not the conventional parameter order
// of test assertion functions where the "expected" value comes first.
func validateRkind(actual reflect.Kind, expected reflect.Kind) error {
	if expected != actual {
		return errors.Errorf(`expected reflect kind %q, got %q`, expected, actual)
	}
	return nil
}

// Same as in `reqdec`.
func sfieldJsonFieldName(sfield reflect.StructField) string {
	return refut.TagIdent(sfield.Tag.Get("json"))
}

// Same as in `gos`.
func sfieldDbColName(sfield reflect.StructField) string {
	return refut.TagIdent(sfield.Tag.Get("db"))
}

func ctxDefault() Ctx {
	return context.Background()
}

func maybeString(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

func last(index int) int { return index - 1 }

// Warning: likely to lead to races. Don't use this even if you THINK you know
// what you're doing.
func isCtxCanceled(ctx Ctx) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

/*
Usage:

	for range counter(N) { ... }
	for i := range counter(N) { ... }

Because `struct{}` is zero-sized, `[]struct{}` is backed by "zerobase" and does
not allocate. This should compile to the same instructions as a "normal"
counted loop.
*/
func counter(n int) []struct{} { return make([]struct{}, n) }
