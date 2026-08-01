package metrics

import "testing"

func TestMustRegister(t *testing.T) {
	MustRegister()

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	MustRegister()
}
