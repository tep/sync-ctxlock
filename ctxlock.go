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

// Package ctxlock provides a stackable, Context-aware locking mechanism that
// allows locks to be freely requested - without blocking - if a parent in the
// call stack already holds the lock. Additionally, since this mechanism is
// Context aware, all blocked goroutines waiting to acquire a lock will be
// resumed when the given Context enters a cancelled state.
//
// The common use case is for a set of methods requiring syncronization where
// one or more may be called either directly (when it would need to acquire the
// lock itself) or indirectly (from another method that has previous acquired
// the lock).
//
// As an example, consider the following construct:
//
//    type Foo struct {
//      ctxlock.ContextLock
//      // other fields
//    }
//
//    func (f *Foo) One(ctx context.Context) (err error) {
//      if ctx, err = f.Lock(ctx); err != nil {
//        return err
//      }
//      defer f.Unlock(ctx)
//
//      // ...do things...
//
//      if f.other() {
//        if err := f.Two(ctx); err != nil {
//          return err
//        }
//        // ...lock is still held here
//      }
//
//      return nil
//    }
//
//    func f *Foo) Two(ctx context.Context) (err error) {
//      if ctx, err = f.Lock(ctx); err != nil {
//        return err
//      }
//      defer f.Unlock(ctx)
//
//      // ...do different things...
//
//      return nil
//    }
//
// In this example, type Foo embeds a ContextLock which is referenced by the
// methods One and Two, both of which may be called directly where each will
// acquire the embedded lock.  However, under certain conditions, One also
// calls Two. This would result in a deadlock if type Foo used sync.Mutex as
// its locking mechanism since Two would block indefinitely waiting to acquire
// the lock held by its caller, method One.
//
// This deadlock is avoided by ContextLock since the Context passed to method
// Two's call to f.Lock is the same Context returned by One's call to f.Lock
// and it informs ContextLock to not block because the lock holder is in the
// current call stack.
//
// When Two is called directly, its deferred call to f.Unlock will release the
// lock.  But when Two is called by One, the ContextLock remains held until One
// itself also returns (making its deferred call to f.Unlock). This locking
// pattern functions as expected to an arbitrary depth.
//
// Note that this pattern remains valid as long as each method in the call
// stack is in the same thread of execution. Synchronization fails for in-stack
// calls to Lock made from goroutines other than the one that originally
// acquired the lock. See the Clear method for how to avoid this problem.
//
package ctxlock

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// ContextLock is a Context aware mutex that avoids blocking if the lock is
// presently held by a parent in its call stack. This behavior is accomplished
// through reference counts tracked in the chain of Contexts provided to and
// returned by calls to ContextLock's methods.
// Note that ContextLock is initialized lazily so its zero value is valid and
// its lock is unheld.
type ContextLock struct {
	cond *sync.Cond
	held bool
	mu   sync.Mutex
	id   lockID
}

type lockID uint64

// Lock locks the receiver c. If the receiver is not currently locked, the lock
// is acquired and this method returns immediately. If the reciever is already
// locked and the provided context indicates that the lock holder is a parent
// in the call stack, Lock will not block but will return immediately with
// a new Context updated to reflect an increased lock reference count. If
// however, the provided context is unaware of the receiver and its lock is
// already held, Lock will block until the lock is released or ctx enters
// a cancelled state.
//
// If the returned error is nil, the lock has been successfully acquired. This
// error is only non-nil if the provided context has entered the cancelled
// state and the error value will be the results from ctx.Err(). Note however
// that regardless of the error value, the returned Context will never be nil.
func (c *ContextLock) Lock(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.id == 0 {
		c._setup()
	}

	depth := c._depth(ctx)

	if depth == 0 {
		for c.held {
			if err := c._wait(ctx); err != nil {
				return ctx, err
			}
		}
		c.held = true
	}

	return c._withDepth(ctx, depth+1)
}

// Unlock decrements the reference count for the receiver's lock. The lock
// will remain held as long as this reference count is greater than zero.
// Once the reference count falls to zero, the lock will be released.
//
// The returned error will only be non-nil if the provided Context has entered
// the cancelled state and the error value will be the results from ctx.Err().
// The returned Context will never be nil.
func (c *ContextLock) Unlock(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.id == 0 {
		return ctx, nil
	}

	depth := c._depth(ctx)

	switch depth {
	case 0:
		if !c.held {
			return ctx, nil
		}
		fallthrough

	case 1:
		c.held = false
		c.cond.Signal()
		fallthrough

	default:
		if depth > 0 {
			depth--
		}
	}

	return c._withDepth(ctx, depth)
}

// Clear checks the receiver's current lock state and, if the lock is currently
// held, returns a copy of ctx with its lock reference count cleared to zero --
// but leaves the receiver in a locked state. If the lock is not currently held,
// ctx itself is returned.
//
// The common use case for this method is when a lock holder needs to spawn new
// goroutines that will execute methods requiring synchronization using the
// receiver's lock. In this situation the lock holder can call Clear to get
// a new Context that can be used in it's spawned goroutines. As long as the
// receiver's lock remains held, the child goroutines will block in their own
// calls to Lock.
//
// The returned error will only be non-nil if the provided Context has entered
// the cancelled state and the error value will be the results from ctx.Err().
// The returned Context will never be nil.
func (c *ContextLock) Clear(ctx context.Context) (context.Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.held {
		return c._withDepth(ctx, 0)
	}

	return ctx, nil
}

func (c *ContextLock) _lock(ctx context.Context) error {
	for c.held {
		if err := c._wait(ctx); err != nil {
			return err
		}
	}

	c.held = true
	return nil
}

func (c *ContextLock) _wait(ctx context.Context) (err error) {
	ch := make(chan struct{})
	defer close(ch)

	go func() {
		select {
		case <-ch:
			return
		case <-ctx.Done():
			err = ctx.Err()
			c.cond.Broadcast()
		}
	}()

	c.cond.Wait()

	return err
}

func (c *ContextLock) _setup() {
	if c.cond == nil {
		c.cond = sync.NewCond(&c.mu)
	}

	for c.id == 0 {
		rand.Seed(time.Now().UnixNano())
		c.id = lockID(rand.Uint64())
	}
}

func (c *ContextLock) _depth(ctx context.Context) int {
	if d, ok := ctx.Value(c.id).(int); ok {
		return d
	}
	return 0
}

func (c *ContextLock) _withDepth(ctx context.Context, value int) (context.Context, error) {
	select {
	case <-ctx.Done():
		c.cond.Broadcast()
		return nil, ctx.Err()
	default:
		return context.WithValue(ctx, c.id, value), nil
	}
}
