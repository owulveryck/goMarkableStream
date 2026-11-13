package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Brute-force protection for /login.
//
// Each client IP gets a few free attempts, then an exponentially growing
// lockout. A global cap on failures per minute bounds the guessing rate even
// for an attacker spreading attempts over many addresses (Funnel exposes the
// login page to the whole internet).
const (
	loginFreeAttempts   = 5
	loginBaseLockout    = time.Second
	loginMaxLockout     = 15 * time.Minute
	loginGlobalWindow   = time.Minute
	loginGlobalMaxFails = 30
	loginMaxTrackedIPs  = 1024
)

type loginAttempts struct {
	fails        int
	blockedUntil time.Time
	lastFail     time.Time
}

type loginLimiter struct {
	mu          sync.Mutex
	perIP       map[string]*loginAttempts
	windowStart time.Time
	windowFails int
	now         func() time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		perIP: make(map[string]*loginAttempts),
		now:   time.Now,
	}
}

// clientIP extracts the peer address. Forwarding headers are ignored on
// purpose: they are attacker-controlled and would let a client pick a fresh
// identity for every attempt.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// allow reports whether ip may attempt a login now; if not, it also returns
// how long the caller should wait.
func (l *loginLimiter) allow(ip string) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()

	if a, ok := l.perIP[ip]; ok && now.Before(a.blockedUntil) {
		return a.blockedUntil.Sub(now), false
	}
	if now.Sub(l.windowStart) < loginGlobalWindow && l.windowFails >= loginGlobalMaxFails {
		return l.windowStart.Add(loginGlobalWindow).Sub(now), false
	}
	return 0, true
}

// failure records a failed attempt from ip.
func (l *loginLimiter) failure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()

	if now.Sub(l.windowStart) >= loginGlobalWindow {
		l.windowStart = now
		l.windowFails = 0
	}
	l.windowFails++

	a, ok := l.perIP[ip]
	if !ok {
		if len(l.perIP) >= loginMaxTrackedIPs {
			l.pruneLocked(now)
		}
		a = &loginAttempts{}
		l.perIP[ip] = a
	}
	a.fails++
	a.lastFail = now
	if a.fails >= loginFreeAttempts {
		lockout := loginBaseLockout << min(a.fails-loginFreeAttempts, 20)
		a.blockedUntil = now.Add(min(lockout, loginMaxLockout))
	}
}

// success forgets the failure history of ip.
func (l *loginLimiter) success(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.perIP, ip)
}

// pruneLocked drops entries that are no longer blocked and have been quiet for
// the maximum lockout period, bounding memory under address spraying.
func (l *loginLimiter) pruneLocked(now time.Time) {
	for ip, a := range l.perIP {
		if now.After(a.blockedUntil) && now.Sub(a.lastFail) > loginMaxLockout {
			delete(l.perIP, ip)
		}
	}
}
