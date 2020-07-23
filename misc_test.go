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

import "fmt"

func formatList(a []byte) string {
	chars := make([]string, len(a))
	for i, c := range a {
		chars[i] = fmt.Sprintf("%q", c)
	}
	return fmt.Sprintf("%v", chars)
}

func permutations(lists [][]byte) [][]byte {
	var out [][]byte
	var perm func(a [][]byte, i int)

	perm = func(a [][]byte, i int) {
		if i > len(a) {
			var s []byte
			for _, x := range a {
				s = append(s, x...)
			}
			out = append(out, s)
			return
		}

		perm(a, i+1)
		for j := i + 1; j < len(a); j++ {
			a[i], a[j] = a[j], a[i]
			perm(a, i+1)
			a[i], a[j] = a[j], a[i]
		}
	}

	perm(lists, 0)

	return out
}
