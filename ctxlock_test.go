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
	"testing"

	"golang.org/x/sync/errgroup"
)

func TestContextLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var tt testThing

	testValues := [][]byte{
		{'A', 'B', 'C'},
		{'L', 'M', 'N', 'O'},
		{'X', 'Y', 'Z'},
	}

	for _, tv := range testValues {
		eg.Go(tt.testfunc(ctx, tv))
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}

	allowed := permutations(testValues)
	if err := tt.check(allowed); err != nil {
		t.Error(err)
		t.Log(" -- wanted one of the following:")
		for _, a := range allowed {
			t.Logf("    %s", formatList(a))
		}
	}
}
