

# toolman.org/sync/ctxlock
`import "toolman.org/sync/ctxlock"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
Package ctxlock provides a stackable, Context-aware locking mechanism that
allows locks to be freely requested - without blocking - if a parent in the
call stack already holds the lock. Additionally, since this mechanism is
Context aware, all blocked goroutines waiting to acquire a lock will be
resumed when the given Context enters a cancelled state.

The common use case is for a set of methods requiring syncronization where
one or more may be called either directly (when it would need to acquire the
lock itself) or indirectly (from another method that has previous acquired
the lock).

As an example, consider the following construct:


	type Foo struct {
	  ctxlock.ContextLock
	  // other fields
	}
	
	func (f *Foo) One(ctx context.Context) (err error) {
	  if ctx, err = f.Lock(ctx); err != nil {
	    return err
	  }
	  defer f.Unlock(ctx)
	
	  // ...do things...
	
	  if f.other() {
	    if err := f.Two(ctx); err != nil {
	      return err
	    }
	    // ...lock is still held here
	  }
	
	  return nil
	}
	
	func f *Foo) Two(ctx context.Context) (err error) {
	  if ctx, err = f.Lock(ctx); err != nil {
	    return err
	  }
	  defer f.Unlock(ctx)
	
	  // ...do different things...
	
	  return nil
	}

In this example, type Foo embeds a ContextLock which is referenced by the
methods One and Two, both of which may be called directly where each will
acquire the embedded lock.  However, under certain conditions, One also
calls Two. This would result in a deadlock if type Foo used sync.Mutex as
its locking mechanism since Two would block indefinitely waiting to acquire
the lock held by its caller, method One.

This deadlock is avoided by ContextLock since the Context passed to method
Two's call to f.Lock is the same Context returned by One's call to f.Lock
and it informs ContextLock to not block because the lock holder is in the
current call stack.

When Two is called directly, its deferred call to f.Unlock will release the
lock.  But when Two is called by One, the ContextLock remains held until One
itself also returns (making its deferred call to f.Unlock). This locking
pattern functions as expected to an arbitrary depth.

Note that this pattern remains valid as long as each method in the call
stack is in the same thread of execution. Synchronization fails for in-stack
calls to Lock made from goroutines other than the one that originally
acquired the lock. See the Clear method for how to avoid this problem.

## <a name="pkg-index">Index</a>
* [type ContextLock](#ContextLock)
  * [func (c *ContextLock) Lock(ctx context.Context) (context.Context, error)](#ContextLock.Lock)
  * [func (c *ContextLock) Unlock(ctx context.Context) (context.Context, error)](#ContextLock.Unlock)
  * [func (c *ContextLock) Clear(ctx context.Context) (context.Context, error)](#ContextLock.Clear)


#### <a name="pkg-files">Package files</a>
[ctxlock.go](/src/target/ctxlock.go) 

## <a name="ContextLock">type</a> [ContextLock](/src/target/ctxlock.go?s=4323:4408#L107)

``` go
type ContextLock struct {
    // contains filtered or unexported fields
}

```
ContextLock is a Context aware mutex that avoids blocking if the lock is
presently held by a parent in its call stack. This behavior is accomplished
through reference counts tracked in the chain of Contexts provided to and
returned by calls to ContextLock's methods.
Note that ContextLock is initialized lazily so its zero value is valid and
its lock is unheld.


### <a name="ContextLock.Lock">func</a> (\*ContextLock) [Lock](/src/target/ctxlock.go?s=5302:5374#L129)

``` go
func (c \*ContextLock) Lock(ctx context.Context) (context.Context, error)
```

Lock locks the receiver c. If the receiver is not currently locked, the lock
is acquired and this method returns immediately. If the reciever is already
locked and the provided context indicates that the lock holder is a parent
in the call stack, Lock will not block but will return immediately with
a new Context updated to reflect an increased lock reference count. If
however, the provided context is unaware of the receiver and its lock is
already held, Lock will block until the lock is released or ctx enters
a cancelled state.

If the returned error is nil, the lock has been successfully acquired. This
error is only non-nil if the provided context has entered the cancelled
state and the error value will be the results from ctx.Err(). Note however
that regardless of the error value, the returned Context will never be nil.


### <a name="ContextLock.Unlock">func</a> (\*ContextLock) [Unlock](/src/target/ctxlock.go?s=6052:6126#L158)

``` go
func (c \*ContextLock) Unlock(ctx context.Context) (context.Context, error)
```

Unlock decrements the reference count for the receiver's lock. The lock
will remain held as long as this reference count is greater than zero.
Once the reference count falls to zero, the lock will be released.

The returned error will only be non-nil if the provided Context has entered
the cancelled state and the error value will be the results from ctx.Err().
The returned Context will never be nil.




### <a name="ContextLock.Clear">func</a> (\*ContextLock) [Clear](/src/target/ctxlock.go?s=7326:7399#L204)
``` go
func (c \*ContextLock) Clear(ctx context.Context) (context.Context, error)
```
Clear checks the receiver's current lock state and, if the lock is currently
held, returns a copy of ctx with its lock reference count cleared to zero --
but leaves the receiver in a locked state. If the lock is not currently held,
ctx itself is returned.

The common use case for this method is when a lock holder needs to spawn new
goroutines that will execute methods requiring synchronization using the
receiver's lock. In this situation the lock holder can call Clear to get
a new Context that can be used in it's spawned goroutines. As long as the
receiver's lock remains held, the child goroutines will block in their own
calls to Lock.

The returned error will only be non-nil if the provided Context has entered
the cancelled state and the error value will be the results from ctx.Err().
The returned Context will never be nil.


