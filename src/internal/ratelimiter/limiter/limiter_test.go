package limiter

import "testing"

func TestCheckLimitConsumesOneCall(t *testing.T) {
	l := New(5)

	allowed, remaining := l.checkLimit()

	if !allowed || remaining != 4 {
		t.Errorf("got (%v, %d), want (true, 4)", allowed, remaining)
	}
}

func TestNewSetsInitialAllowance(t *testing.T) {
	for _, initial := range []int{0, 1, 5} {
		l := New(initial)
		if l == nil {
			t.Fatalf("New(%d) returned nil", initial)
		}
		if l.remaining != initial {
			t.Errorf("New(%d).remaining = %d, want %d", initial, l.remaining, initial)
		}
	}
}

func TestCheckLimitExhaustion(t *testing.T) {
	l := New(3)
	for call := 1; call <= 6; call++ {
		allowed, remaining := l.checkLimit()
		wantAllowed := call <= 3
		wantRemaining := max(3-call, 0)
		if allowed != wantAllowed || remaining != wantRemaining {
			t.Errorf("call %d: got (%v, %d), want (%v, %d)", call, allowed, remaining, wantAllowed, wantRemaining)
		}
	}
}

func TestCheckLimitDeniesWithZeroAllowance(t *testing.T) {
	l := New(0)
	for call := 1; call <= 3; call++ {
		allowed, remaining := l.checkLimit()
		if allowed || remaining != 0 {
			t.Errorf("call %d: got (%v, %d), want (false, 0)", call, allowed, remaining)
		}
	}
}

func TestNewLimitersAreIndependent(t *testing.T) {
	first, second := New(5), New(5)
	first.checkLimit()
	allowed, remaining := second.checkLimit()
	if !allowed || remaining != 4 {
		t.Errorf("second limiter got (%v, %d), want (true, 4)", allowed, remaining)
	}
}

func TestNewUserLimitersInitializesEmptyMap(t *testing.T) {
	users := NewUserLimiters()
	if users == nil || users.users == nil {
		t.Fatal("constructor must return a collection with an initialized map")
	}
	if len(users.users) != 0 {
		t.Errorf("new collection has %d entries, want 0", len(users.users))
	}
}

func TestForUsersCreatesAndReusesLimiter(t *testing.T) {
	users := NewUserLimiters()
	l := users.forUsers("1")
	if l == nil {
		t.Fatal("ForUsers returned nil")
	}
	if l.remaining != 5 {
		t.Errorf("new user's allowance = %d, want 5", l.remaining)
	}
	l.checkLimit()
	if got := users.forUsers("1"); got != l {
		t.Fatal("repeated lookup returned a different limiter")
	}
	if l.remaining != 4 {
		t.Errorf("lookup reset allowance to %d, want 4", l.remaining)
	}
	if len(users.users) != 1 {
		t.Errorf("collection has %d entries, want 1", len(users.users))
	}
}

func TestForUsersKeepsExhaustedLimiter(t *testing.T) {
	users := NewUserLimiters()
	l := users.forUsers("1")
	for call := 0; call < 5; call++ {
		l.checkLimit()
	}
	again := users.forUsers("1")
	if again != l {
		t.Fatal("exhausted limiter was replaced")
	}
	allowed, remaining := again.checkLimit()
	if allowed || remaining != 0 {
		t.Errorf("exhausted user got (%v, %d), want (false, 0)", allowed, remaining)
	}
}

func TestForUsersKeepsUsersIndependent(t *testing.T) {
	users := NewUserLimiters()
	first, second := users.forUsers("1"), users.forUsers("2")
	if first == second {
		t.Fatal("different users share the same limiter")
	}
	for call := 0; call < 5; call++ {
		first.checkLimit()
	}
	allowed, remaining := second.checkLimit()
	if !allowed || remaining != 4 {
		t.Errorf("second user got (%v, %d), want (true, 4)", allowed, remaining)
	}
}

func TestUserCollectionsAreIndependent(t *testing.T) {
	first, second := NewUserLimiters(), NewUserLimiters()
	first.forUsers("1").checkLimit()
	allowed, remaining := second.forUsers("1").checkLimit()
	if !allowed || remaining != 4 {
		t.Errorf("second collection got (%v, %d), want (true, 4)", allowed, remaining)
	}
}

func TestAllowUsesPerUserAllowance(t *testing.T) {
	users := NewUserLimiters()
	for _, userID := range []string{"alice", "bob"} {
		for call := 1; call <= 7; call++ {
			if got, want := users.Allow(userID), call <= 5; got != want {
				t.Errorf("user %q, call %d: Allow = %v, want %v", userID, call, got, want)
			}
		}
	}
	if users.Allow("alice") {
		t.Error("exhausted user received a fresh allowance after another user's requests")
	}
}
