package limiter

import "testing"

func TestCheckLimitConsumesOneCall(t *testing.T) {
	l := New(5)

	allowed, remaining := l.CheckLimit()

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
		allowed, remaining := l.CheckLimit()
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
		allowed, remaining := l.CheckLimit()
		if allowed || remaining != 0 {
			t.Errorf("call %d: got (%v, %d), want (false, 0)", call, allowed, remaining)
		}
	}
}

func TestNewLimitersAreIndependent(t *testing.T) {
	first, second := New(5), New(5)
	first.CheckLimit()
	allowed, remaining := second.CheckLimit()
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
	l := users.ForUsers("1")
	if l == nil {
		t.Fatal("ForUsers returned nil")
	}
	if l.remaining != 5 {
		t.Errorf("new user's allowance = %d, want 5", l.remaining)
	}
	l.CheckLimit()
	if got := users.ForUsers("1"); got != l {
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
	l := users.ForUsers("1")
	for call := 0; call < 5; call++ {
		l.CheckLimit()
	}
	again := users.ForUsers("1")
	if again != l {
		t.Fatal("exhausted limiter was replaced")
	}
	allowed, remaining := again.CheckLimit()
	if allowed || remaining != 0 {
		t.Errorf("exhausted user got (%v, %d), want (false, 0)", allowed, remaining)
	}
}

func TestForUsersKeepsUsersIndependent(t *testing.T) {
	users := NewUserLimiters()
	first, second := users.ForUsers("1"), users.ForUsers("2")
	if first == second {
		t.Fatal("different users share the same limiter")
	}
	for call := 0; call < 5; call++ {
		first.CheckLimit()
	}
	allowed, remaining := second.CheckLimit()
	if !allowed || remaining != 4 {
		t.Errorf("second user got (%v, %d), want (true, 4)", allowed, remaining)
	}
}

func TestUserCollectionsAreIndependent(t *testing.T) {
	first, second := NewUserLimiters(), NewUserLimiters()
	first.ForUsers("1").CheckLimit()
	allowed, remaining := second.ForUsers("1").CheckLimit()
	if !allowed || remaining != 4 {
		t.Errorf("second collection got (%v, %d), want (true, 4)", allowed, remaining)
	}
}
