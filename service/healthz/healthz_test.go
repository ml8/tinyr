package healthz

import (
	"context"
	"errors"
	"testing"
	"time"
)

type delay int

func (d delay) Healthz(ctx context.Context) error {
	time.Sleep(time.Duration(d) * time.Second)
	return nil
}

type mockcheck struct {
	fail   bool
	called int
}

func (c *mockcheck) Healthz(ctx context.Context) error {
	c.called = c.called + 1
	if c.fail {
		return errors.New("uh-oh")
	}
	return nil
}

func mock(fail bool) *mockcheck {
	return &mockcheck{fail: fail, called: 0}
}

func TestTimeout(t *testing.T) {
	SetDeadline(500 * time.Millisecond)
	Register(delay(1))
	err := Healthz()
	if err == nil {
		t.Errorf("Deadline should've been exceeded!")
	} else if _, ok := err.(DeadlineExceeded); !ok {
		t.Errorf("Incorrect error type %+v", err)
	}
}

func TestNoTimeout(t *testing.T) {
	SetDeadline(0)
	Register(delay(1))
	err := Healthz()
	if err != nil {
		t.Errorf("Should not have timed out!")
	}
}

func TestOk(t *testing.T) {
	var checks []Component
	checks = append(checks, mock(false))
	checks = append(checks, mock(false))
	checks = append(checks, mock(false))

	SetDeadline(0)
	Register(checks...)

	err := Healthz()
	if err != nil {
		t.Errorf("Should not have failed")
	}

	for i := range checks {
		count := checks[i].(*mockcheck).called
		if count != 1 {
			t.Errorf("Incorrect call count for idx %v: %v", i, count)
		}
	}
}

func TestFail(t *testing.T) {
	var checks []Component
	checks = append(checks, mock(false))
	checks = append(checks, mock(true))
	checks = append(checks, mock(false))

	expected := []int{1, 1, 0}

	SetDeadline(0)
	Register(checks...)

	err := Healthz()
	if err == nil {
		t.Errorf("Should have failed")
	}

	for i := range checks {
		count := checks[i].(*mockcheck).called
		if count != expected[i] {
			t.Errorf("Incorrect call count for idx %v: %v", i, count)
		}
	}
}
