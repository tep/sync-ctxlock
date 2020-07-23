// Copyright 2020 Timothy E. Peoples
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
// IN THE SOFTWARE.
//
// ----------------------------------------------------------------------------

package ctxlock

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

type testThing struct {
	values []byte
	ContextLock
}

func (tt *testThing) testfunc(ctx context.Context, values []byte) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
		defer cancel()

		return tt.shiftNext(ctx, values)
	}
}

func (tt *testThing) check(allowed [][]byte) error {
	for _, want := range allowed {
		if reflect.DeepEqual(tt.values, want) {
			return nil
		}
	}

	return fmt.Errorf("values out of order; got %v", formatList(tt.values))
}

func (tt *testThing) shiftEven(ctx context.Context, values []byte) (err error) {
	if ctx, err = tt.Lock(ctx); err != nil {
		return err
	}
	defer tt.Unlock(ctx)

	tt.values = append(tt.values, values[0])
	return tt.shiftNext(ctx, values[1:])
}

func (tt *testThing) shiftOdd(ctx context.Context, values []byte) (err error) {
	if ctx, err = tt.Lock(ctx); err != nil {
		return err
	}
	defer tt.Unlock(ctx)

	tt.values = append(tt.values, values[0])
	return tt.shiftNext(ctx, values[1:])
}

func (tt *testThing) shiftNext(ctx context.Context, values []byte) error {
	if len(values) == 0 {
		return nil
	}

	if values[0]%2 == 1 {
		return tt.shiftOdd(ctx, values)
	}
	return tt.shiftEven(ctx, values)
}
