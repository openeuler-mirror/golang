// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || plan9 || solaris || windows) && goexperiment.spinbitmutex

package runtime

import (
	"internal/goarch"
	"internal/runtime/atomic"
	"unsafe"
)

// This implementation depends on OS-specific implementations of
//
//	func semacreate(mp *m)
//		Create a semaphore for mp, if it does not already have one.
//
//	func semasleep(ns int64) int32
//		If ns < 0, acquire m's semaphore and return 0.
//		If ns >= 0, try to acquire m's semaphore for at most ns nanoseconds.
//		Return 0 if the semaphore was acquired, -1 if interrupted or timed out.
//
//	func semawakeup(mp *m)
//		Wake up mp, which is or will soon be sleeping on its semaphore.

// The mutex state consists of four flags and a pointer. The flag at bit 0,
// mutexLocked, represents the lock itself. Bit 1, mutexSleeping, is a hint that
// the pointer is non-nil. The fast paths for locking and unlocking the mutex
// are based on atomic 8-bit swap operations on the low byte; bits 2 through 7
// are unused.
//
// Bit 8, mutexSpinning, is a try-lock that grants a waiting M permission to
// spin on the state word. Most other Ms must attempt to spend their time
// sleeping to reduce traffic on the cache line. This is the "spin bit" for
// which the implementation is named. (The anti-starvation mechanism also grants
// temporary permission for an M to spin.)
//
// Bit 9, mutexStackLocked, is a try-lock that grants an unlocking M permission
// to inspect the list of waiting Ms and to pop an M off of that stack.
//
// The upper bits hold a (partial) pointer to the M that most recently went to
// sleep. The sleeping Ms form a stack linked by their mWaitList.next fields.
// Because the fast paths use an 8-bit swap on the low byte of the state word,
// we'll need to reconstruct the full M pointer from the bits we have. Most Ms
// are allocated on the heap, and have a known alignment and base offset. (The
// offset is due to mallocgc's allocation headers.) The main program thread uses
// a static M value, m0. We check for m0 specifically and add a known offset
// otherwise.

const (
	active_spin     = 4  // referenced in proc.go for sync.Mutex implementation
	active_spin_cnt = 30 // referenced in proc.go for sync.Mutex implementation
)

const (
	mutexLocked      = 0x001
	mutexSleeping    = 0x002
	mutexSpinning    = 0x100
	mutexStackLocked = 0x200
	mutexMMask       = 0x3FF
	mutexMOffset     = mallocHeaderSize // alignment of heap-allocated Ms (those other than m0)

	mutexActiveSpinCount  = 4
	mutexActiveSpinSize   = 30
	mutexPassiveSpinCount = 1

	mutexTailWakePeriod = 16

	// Adaptive spin tuning parameters (arm64 only).
	// These control how the per-goroutine spin size is adjusted based on
	// observed lock acquisition behavior.
	adaptiveSpinProcyieldThreshold = 1000 // procyield count to trigger adjustment
	adaptiveSpinOsyieldHigh        = 200  // osyield count: indicates relatively long critical section
	adaptiveSpinOsyieldMedium      = 50   // osyield count: indicates relatively short critical section
	adaptiveSpinSizeMin            = 30   // minimum spin size (default)
	adaptiveSpinSizeMax            = 1000 // maximum spin size
	adaptiveSpinAdjustStep         = 90   // step size for spin adjustment
	adaptiveSpinSemaThreshold      = 500  // sema count: reflects lock contention degree
	adaptiveSpinAcqHigh            = 400  // lock acquisitions during spin (We + WeNoSleep): high threshold
	adaptiveSpinAcqLow             = 300  // lock acquisitions during spin (We + WeNoSleep): low threshold
)

// mutexAdaptiveSpinEnabled returns whether the adaptive spin mode is enabled.
// -gcflags=-d=mutexadaptivespin=[0|1] (default 0, disabled).
func mutexAdaptiveSpinEnabled() uint32 { return 0 }

//go:nosplit
func key8(p *uintptr) *uint8 {
	if goarch.BigEndian {
		return &(*[8]uint8)(unsafe.Pointer(p))[goarch.PtrSize/1-1]
	}
	return &(*[8]uint8)(unsafe.Pointer(p))[0]
}

// mWaitList is part of the M struct, and holds the list of Ms that are waiting
// for a particular runtime.mutex.
//
// When an M is unable to immediately obtain a lock, it adds itself to the list
// of Ms waiting for the lock. It does that via this struct's next field,
// forming a singly-linked list with the mutex's key field pointing to the head
// of the list.
type mWaitList struct {
	next muintptr // next m waiting for lock
}

// lockVerifyMSize confirms that we can recreate the low bits of the M pointer.
func lockVerifyMSize() {
	size := roundupsize(unsafe.Sizeof(m{}), false) + mallocHeaderSize
	if size&mutexMMask != 0 {
		print("M structure uses sizeclass ", size, "/", hex(size), " bytes; ",
			"incompatible with mutex flag mask ", hex(mutexMMask), "\n")
		throw("runtime.m memory alignment too small for spinbit mutex")
	}
}

// mutexWaitListHead recovers a full muintptr that was missing its low bits.
// With the exception of the static m0 value, it requires allocating runtime.m
// values in a size class with a particular minimum alignment. The 2048-byte
// size class allows recovering the full muintptr value even after overwriting
// the low 11 bits with flags. We can use those 11 bits as 3 flags and an
// atomically-swapped byte.
//
//go:nosplit
func mutexWaitListHead(v uintptr) muintptr {
	if highBits := v &^ mutexMMask; highBits == 0 {
		return 0
	} else if m0bits := muintptr(unsafe.Pointer(&m0)); highBits == uintptr(m0bits)&^mutexMMask {
		return m0bits
	} else {
		return muintptr(highBits + mutexMOffset)
	}
}

// mutexPreferLowLatency reports if this mutex prefers low latency at the risk
// of performance collapse. If so, we can allow all waiting threads to spin on
// the state word rather than go to sleep.
//
// TODO: We could have the waiting Ms each spin on their own private cache line,
// especially if we can put a bound on the on-CPU time that would consume.
//
// TODO: If there's a small set of mutex values with special requirements, they
// could make use of a more specialized lock2/unlock2 implementation. Otherwise,
// we're constrained to what we can fit within a single uintptr with no
// additional storage on the M for each lock held.
//
//go:nosplit
func mutexPreferLowLatency(l *mutex) bool {
	switch l {
	default:
		return false
	case &sched.lock:
		// We often expect sched.lock to pass quickly between Ms in a way that
		// each M has unique work to do: for instance when we stop-the-world
		// (bringing each P to idle) or add new netpoller-triggered work to the
		// global run queue.
		return true
	}
}

func mutexContended(l *mutex) bool {
	return atomic.Loaduintptr(&l.key) > mutexLocked
}

// adaptiveSpinActive reports whether the adaptive spin mode is enabled.
func adaptiveSpinActive() bool {
	return goarch.IsArm64 == 1 && mutexAdaptiveSpinEnabled() == 1
}

// adaptiveSpinAccountAcq records a lock acquisition made while spinning.
func adaptiveSpinAccountAcq(gp *g, sleepFlag bool) {
	if adaptiveSpinActive() {
		if sleepFlag {
			gp.lockSpinAcqWe++
		} else {
			gp.lockSpinAcqWeNoSleep++
		}
	}
}

// adaptiveSpinAccountAcqNoWe records a lock acquisition made without spinning.
func adaptiveSpinAccountAcqNoWe(gp *g, sleepFlag bool) {
	if adaptiveSpinActive() && sleepFlag {
		gp.lockSpinAcqNoWe++
	}
}

// adaptiveSpinAccountSleep records a sleep while contending for the lock,
// returning the updated sleep flag.
func adaptiveSpinAccountSleep(gp *g, sleepFlag bool) bool {
	if adaptiveSpinActive() {
		gp.lockSemaCnt++
		return true
	}
	return sleepFlag
}

// adaptiveProcyield actively spins on the lock with the per-goroutine
// adaptive spin size.
func adaptiveProcyield(gp *g) {
	if gp.adaptiveSpinSize == 0 {
		gp.adaptiveSpinSize = mutexActiveSpinSize
	}
	gp.lockProcyieldCnt++
	procyield(gp.adaptiveSpinSize)
}

// adaptiveOsyield passively spins, adjusting the adaptive spin size based on
// the observed lock acquisition behavior.
func adaptiveOsyield(gp *g) {
	gp.lockOsyieldCnt++
	adaptiveSpinAdjust(gp)
}

// adaptiveSpinAdjust tunes the adaptive spin size once lockProcyieldCnt
// exceeds adaptiveSpinProcyieldThreshold, then resets the counters.
func adaptiveSpinAdjust(gp *g) {
	if gp.lockProcyieldCnt <= adaptiveSpinProcyieldThreshold {
		return
	}
	if gp.lockOsyieldCnt > adaptiveSpinOsyieldHigh {
		// Long critical sections: spinning has little value.
		gp.adaptiveSpinSize = adaptiveSpinSizeMin
	} else if gp.lockOsyieldCnt > adaptiveSpinOsyieldMedium {
		adaptiveSpinAdjustByContention(gp)
	}
	gp.lockProcyieldCnt = 0
	gp.lockOsyieldCnt = 0
	gp.lockSpinAcqWe = 0
	gp.lockSpinAcqNoWe = 0
	gp.lockSemaCnt = 0
	gp.lockSpinAcqWeNoSleep = 0
}

// adaptiveSpinAdjustByContention tunes the adaptive spin size based on the
// observed lock contention.
func adaptiveSpinAdjustByContention(gp *g) {
	if gp.lockSemaCnt < adaptiveSpinSemaThreshold {
		// Low contention: tune the spin size to how often the lock was
		// acquired while spinning.
		if gp.lockSpinAcqWe+gp.lockSpinAcqWeNoSleep > adaptiveSpinAcqHigh {
			gp.adaptiveSpinSize = adaptiveSpinIncrease(gp.adaptiveSpinSize)
		} else if gp.lockSpinAcqWe+gp.lockSpinAcqWeNoSleep < adaptiveSpinAcqLow {
			gp.adaptiveSpinSize = adaptiveSpinSizeMin
		}
		return
	}
	// High contention: tune by the ratio of acquisitions to sleeps.
	if (gp.lockSpinAcqWe+gp.lockSpinAcqNoWe)*2 < gp.lockSemaCnt {
		gp.adaptiveSpinSize = adaptiveSpinIncrease(gp.adaptiveSpinSize)
	} else if (gp.lockSpinAcqWe+gp.lockSpinAcqNoWe)*4 > gp.lockSemaCnt*3 {
		gp.adaptiveSpinSize = adaptiveSpinDecrease(gp.adaptiveSpinSize)
	}
}

// adaptiveSpinIncrease returns size plus adaptiveSpinAdjustStep, or size when
// that would exceed adaptiveSpinSizeMax.
func adaptiveSpinIncrease(size uint32) uint32 {
	if newSize := size + adaptiveSpinAdjustStep; newSize <= adaptiveSpinSizeMax {
		return newSize
	}
	return size
}

// adaptiveSpinDecrease returns size minus adaptiveSpinAdjustStep, or size when
// it is not greater than adaptiveSpinAdjustStep.
func adaptiveSpinDecrease(size uint32) uint32 {
	if size > adaptiveSpinAdjustStep {
		return size - adaptiveSpinAdjustStep
	}
	return size
}

func lock(l *mutex) {
	lockWithRank(l, getLockRank(l))
}

func lock2(l *mutex) {
	gp := getg()
	if gp.m.locks < 0 {
		throw("runtime·lock: lock count")
	}
	gp.m.locks++

	k8 := key8(&l.key)

	// Speculative grab for lock.
	v8 := atomic.Xchg8(k8, mutexLocked)
	if v8&mutexLocked == 0 {
		if v8&mutexSleeping != 0 {
			atomic.Or8(k8, mutexSleeping)
		}
		return
	}
	semacreate(gp.m)

	timer := &lockTimer{lock: l}
	timer.begin()
	// On uniprocessors, no point spinning.
	// On multiprocessors, spin for mutexActiveSpinCount attempts.
	spin := 0
	if ncpu > 1 {
		spin = mutexActiveSpinCount
	}
	var weSpin, atTail, sleepFlag bool
	v := atomic.Loaduintptr(&l.key)
tryAcquire:
	for i := 0; ; i++ {
		if v&mutexLocked == 0 {
			if weSpin {
				next := (v &^ mutexSpinning) | mutexSleeping | mutexLocked
				if next&^mutexMMask == 0 {
					// The fast-path Xchg8 may have cleared mutexSleeping. Fix
					// the hint so unlock2 knows when to use its slow path.
					next = next &^ mutexSleeping
				}
				if atomic.Casuintptr(&l.key, v, next) {
					timer.end()
					adaptiveSpinAccountAcq(gp, sleepFlag)
					return
				}
			} else {
				prev8 := atomic.Xchg8(k8, mutexLocked|mutexSleeping)
				if prev8&mutexLocked == 0 {
					timer.end()
					adaptiveSpinAccountAcqNoWe(gp, sleepFlag)
					return
				}
			}
			v = atomic.Loaduintptr(&l.key)
			continue tryAcquire
		}

		if !weSpin && v&mutexSpinning == 0 && atomic.Casuintptr(&l.key, v, v|mutexSpinning) {
			v |= mutexSpinning
			weSpin = true
		}

		if weSpin || atTail || mutexPreferLowLatency(l) {
			if i < spin {
				if adaptiveSpinActive() {
					adaptiveProcyield(gp)
				} else {
					procyield(mutexActiveSpinSize)
				}
				v = atomic.Loaduintptr(&l.key)
				continue tryAcquire
			} else if i < spin+mutexPassiveSpinCount {
				if adaptiveSpinActive() {
					adaptiveOsyield(gp)
				}
				osyield()
				v = atomic.Loaduintptr(&l.key)
				continue tryAcquire
			}
		}

		// Go to sleep
		if v&mutexLocked == 0 {
			throw("runtime·lock: sleeping while lock is available")
		}

		// Store the current head of the list of sleeping Ms in our gp.m.mWaitList.next field
		gp.m.mWaitList.next = mutexWaitListHead(v)

		// Pack a (partial) pointer to this M with the current lock state bits
		next := (uintptr(unsafe.Pointer(gp.m)) &^ mutexMMask) | v&mutexMMask | mutexSleeping
		if weSpin { // If we were spinning, prepare to retire
			next = next &^ mutexSpinning
		}

		if atomic.Casuintptr(&l.key, v, next) {
			weSpin = false
			// We've pushed ourselves onto the stack of waiters. Wait.
			semasleep(-1)
			atTail = gp.m.mWaitList.next == 0 // we were at risk of starving
			i = 0
			sleepFlag = adaptiveSpinAccountSleep(gp, sleepFlag)
		}
		gp.m.mWaitList.next = 0
		v = atomic.Loaduintptr(&l.key)
	}
}

func unlock(l *mutex) {
	unlockWithRank(l)
}

// We might not be holding a p in this code.
//
//go:nowritebarrier
func unlock2(l *mutex) {
	gp := getg()

	prev8 := atomic.Xchg8(key8(&l.key), 0)
	if prev8&mutexLocked == 0 {
		throw("unlock of unlocked lock")
	}

	if prev8&mutexSleeping != 0 {
		unlock2Wake(l)
	}

	gp.m.mLockProfile.recordUnlock(l)
	gp.m.locks--
	if gp.m.locks < 0 {
		throw("runtime·unlock: lock count")
	}
	if gp.m.locks == 0 && gp.preempt { // restore the preemption request in case we've cleared it in newstack
		gp.stackguard0 = stackPreempt
	}
}

// unlock2Wake updates the list of Ms waiting on l, waking an M if necessary.
//
//go:nowritebarrier
func unlock2Wake(l *mutex) {
	v := atomic.Loaduintptr(&l.key)

	// On occasion, seek out and wake the M at the bottom of the stack so it
	// doesn't starve.
	antiStarve := cheaprandn(mutexTailWakePeriod) == 0
	if !(antiStarve || // avoiding starvation may require a wake
		v&mutexSpinning == 0 || // no spinners means we must wake
		mutexPreferLowLatency(l)) { // prefer waiters be awake as much as possible
		return
	}

	for {
		if v&^mutexMMask == 0 || v&mutexStackLocked != 0 {
			// No waiting Ms means nothing to do.
			//
			// If the stack lock is unavailable, its owner would make the same
			// wake decisions that we would, so there's nothing for us to do.
			//
			// Although: This thread may have a different call stack, which
			// would result in a different entry in the mutex contention profile
			// (upon completion of go.dev/issue/66999). That could lead to weird
			// results if a slow critical section ends but another thread
			// quickly takes the lock, finishes its own critical section,
			// releases the lock, and then grabs the stack lock. That quick
			// thread would then take credit (blame) for the delay that this
			// slow thread caused. The alternative is to have more expensive
			// atomic operations (a CAS) on the critical path of unlock2.
			return
		}
		// Other M's are waiting for the lock.
		// Obtain the stack lock, and pop off an M.
		next := v | mutexStackLocked
		if atomic.Casuintptr(&l.key, v, next) {
			break
		}
		v = atomic.Loaduintptr(&l.key)
	}

	// We own the mutexStackLocked flag. New Ms may push themselves onto the
	// stack concurrently, but we're now the only thread that can remove or
	// modify the Ms that are sleeping in the list.

	var committed *m // If we choose an M within the stack, we've made a promise to wake it
	for {
		headM := v &^ mutexMMask
		flags := v & (mutexMMask &^ mutexStackLocked) // preserve low bits, but release stack lock

		mp := mutexWaitListHead(v).ptr()
		wakem := committed
		if committed == nil {
			if v&mutexSpinning == 0 || mutexPreferLowLatency(l) {
				wakem = mp
			}
			if antiStarve {
				// Wake the M at the bottom of the stack of waiters. (This is
				// O(N) with the number of waiters.)
				wakem = mp
				prev := mp
				for {
					next := wakem.mWaitList.next.ptr()
					if next == nil {
						break
					}
					prev, wakem = wakem, next
				}
				if wakem != mp {
					prev.mWaitList.next = wakem.mWaitList.next
					committed = wakem
				}
			}
		}

		if wakem == mp {
			headM = uintptr(mp.mWaitList.next) &^ mutexMMask
		}

		next := headM | flags
		if atomic.Casuintptr(&l.key, v, next) {
			if wakem != nil {
				// Claimed an M. Wake it.
				semawakeup(wakem)
			}
			break
		}

		v = atomic.Loaduintptr(&l.key)
	}
}
