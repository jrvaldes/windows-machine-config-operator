package locker

import "sync"

// ReconcileLocker manages exclusive locks for reconcile operations
type ReconcileLocker struct {
	lock *sync.Mutex
}

func NewReconcileLocker() *ReconcileLocker {
	return &ReconcileLocker{
		lock: new(sync.Mutex),
	}
}

func (rl *ReconcileLocker) Lock() {
	rl.lock.Lock()
}

func (rl *ReconcileLocker) Unlock() {
	rl.lock.Unlock()
}
