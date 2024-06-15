package healthz

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	mu     sync.Mutex
	checks []Component
	dl     time.Duration
)

type DeadlineExceeded time.Duration

func (d DeadlineExceeded) Error() string {
	return fmt.Sprintf("Deadline exeeded (%v)", time.Duration(d))
}

// Set timeout for health checks.
func SetDeadline(deadline time.Duration) {
	dl = deadline
}

// Register a component for health checking. Components will be checked in the
// order registered.
func Register(components ...Component) {
	mu.Lock()
	defer mu.Unlock()
	checks = append(checks, components...)
}

// Component to be checked.
type Component interface {
	// Return true iff the component is healthy. Component not required to check
	// for context deadline; calling code will do so.
	Healthz(ctx context.Context) error
}

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := Healthz()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte("ok"))
	}
}

// Check all components.
func Healthz() (err error) {
	mu.Lock()
	defer mu.Unlock()
	ctx := context.Background()
	if dl > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), dl)
		defer cancel()
	}
	if err = checkAllLocked(ctx); err != nil {
		return
	}
	return
}

// Check all components.
func CheckAll(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()
	return checkAllLocked(ctx)
}

func checkAllLocked(ctx context.Context) (err error) {
	for _, c := range checks {
		err = c.Healthz(ctx)
		select {
		case <-ctx.Done():
			err = DeadlineExceeded(dl)
		default:
		}
		if err != nil {
			break
		}
	}
	return
}
