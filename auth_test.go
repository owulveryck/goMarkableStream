package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/owulveryck/goMarkableStream/internal/jwtutil"
)

func newTestJWTManager(t *testing.T) *jwtutil.Manager {
	t.Helper()
	mgr := jwtutil.NewManager(jwtutil.ManagerConfig{SecretDir: t.TempDir(), TokenLifetime: time.Hour, AutoGenerate: true})
	if err := mgr.Initialize(); err != nil {
		t.Fatal(err)
	}
	return mgr
}

// withAuthGlobals swaps the package-level config and funnel credentials for
// the duration of a test.
func withAuthGlobals(t *testing.T, user, pass string) *FunnelCredentials {
	t.Helper()
	oldC, oldFC := c, funnelCreds
	c.Username, c.Password = user, pass
	funnelCreds = NewFunnelCredentials()
	t.Cleanup(func() { c, funnelCreds = oldC, oldFC })
	return funnelCreds
}

func TestCheckCredentialsRoles(t *testing.T) {
	fc := withAuthGlobals(t, "owner", "s3cret!")
	if role, _, ok := checkCredentials("owner", "s3cret!"); !ok || role != jwtutil.RoleAdmin {
		t.Errorf("owner: role=%q ok=%v, want admin", role, ok)
	}
	if _, _, ok := checkCredentials("owner", "wrong"); ok {
		t.Error("wrong password accepted")
	}
	u, p := fc.Generate()
	role, gen, ok := checkCredentials(u, p)
	if !ok || role != jwtutil.RoleFunnel || gen == 0 {
		t.Errorf("funnel: role=%q gen=%d ok=%v", role, gen, ok)
	}
}

// protectedProbe records the role AuthMiddleware put in the request context.
func protectedProbe(got *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = requestRole(r)
		w.WriteHeader(http.StatusOK)
	})
}

func doAuth(h http.Handler, token string) int {
	req := httptest.NewRequest(http.MethodGet, "/stream", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestAuthMiddlewareRoles(t *testing.T) {
	fc := withAuthGlobals(t, "owner", "s3cret!")
	mgr := newTestJWTManager(t)
	var role string
	h := AuthMiddleware(protectedProbe(&role), mgr)

	admin, _ := mgr.CreateScopedToken("owner", jwtutil.RoleAdmin, 0)
	if code := doAuth(h, admin); code != http.StatusOK || role != jwtutil.RoleAdmin {
		t.Errorf("admin token: code=%d role=%q", code, role)
	}

	u, p := fc.Generate()
	_, gen, _ := checkCredentials(u, p)
	viewer, _ := mgr.CreateScopedToken(u, jwtutil.RoleFunnel, gen)
	if code := doAuth(h, viewer); code != http.StatusOK || role != jwtutil.RoleFunnel {
		t.Errorf("funnel token: code=%d role=%q", code, role)
	}

	// Disabling Funnel must revoke viewer tokens...
	fc.Clear()
	if code := doAuth(h, viewer); code != http.StatusUnauthorized {
		t.Errorf("funnel token after Clear: code=%d, want 401", code)
	}
	// ...and so must re-enabling it with fresh credentials.
	fc.Generate()
	if code := doAuth(h, viewer); code != http.StatusUnauthorized {
		t.Errorf("funnel token after regeneration: code=%d, want 401", code)
	}

	// Role-less tokens (issued before roles existed) are refused.
	legacy, _ := jwtutil.SignToken(jwtutil.Claims{Subject: "stream", ExpiresAt: time.Now().Add(time.Hour).Unix()}, mustSecret(t, mgr))
	if code := doAuth(h, legacy); code != http.StatusUnauthorized {
		t.Errorf("role-less token: code=%d, want 401", code)
	}
	if code := doAuth(h, ""); code != http.StatusUnauthorized {
		t.Errorf("no token: code=%d, want 401", code)
	}
}

func mustSecret(t *testing.T, mgr *jwtutil.Manager) []byte {
	t.Helper()
	secret, err := mgr.GetStore().Load()
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func TestRequestRoleWithoutMiddlewareIsAdmin(t *testing.T) {
	// -unsafe mode bypasses AuthMiddleware entirely.
	if got := requestRole(httptest.NewRequest(http.MethodGet, "/funnel", nil)); got != jwtutil.RoleAdmin {
		t.Errorf("requestRole = %q, want admin", got)
	}
}

func TestUsingDefaultCredentials(t *testing.T) {
	for pass, want := range map[string]bool{"": true, "password": true, "correct horse": false} {
		if got := usingDefaultCredentials(&configuration{Password: pass}); got != want {
			t.Errorf("password %q: got %v, want %v", pass, got, want)
		}
	}
}

func TestLoginLimiter(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	l := newLoginLimiter()
	l.now = func() time.Time { return now }

	for i := range loginFreeAttempts {
		if _, ok := l.allow("1.2.3.4"); !ok {
			t.Fatalf("attempt %d blocked too early", i+1)
		}
		l.failure("1.2.3.4")
	}
	if _, ok := l.allow("1.2.3.4"); ok {
		t.Fatal("not locked out after free attempts")
	}
	if _, ok := l.allow("5.6.7.8"); !ok {
		t.Error("lockout leaked to another IP")
	}
	now = now.Add(2 * loginBaseLockout)
	if _, ok := l.allow("1.2.3.4"); !ok {
		t.Error("still locked after lockout expired")
	}
	l.success("1.2.3.4")
	if _, ok := l.perIP["1.2.3.4"]; ok {
		t.Error("success did not reset history")
	}

	// Global cap: failures spread over many IPs still hit a ceiling.
	now = now.Add(loginGlobalWindow)
	for i := range loginGlobalMaxFails {
		l.failure(string(rune('a' + i)))
	}
	if _, ok := l.allow("9.9.9.9"); ok {
		t.Error("global failure cap not enforced")
	}
	now = now.Add(loginGlobalWindow)
	if _, ok := l.allow("9.9.9.9"); !ok {
		t.Error("global cap not reset after window")
	}
}
