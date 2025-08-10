package runtime

import (
	"testing"

	"GoWorkerAI/app/tools"
)

func TestRuntimeAddTools(t *testing.T) {
	r := NewRuntime(nil, nil, nil, nil, false)
	r.AddTools([]tools.Tool{{Name: "a"}, {Name: "b"}})
	tk := r.Toolkit()
	if len(tk) != 2 || tk["a"].Name != "a" || tk["b"].Name != "b" {
		t.Fatalf("unexpected toolkit: %#v", tk)
	}
	tk["a"] = tools.Tool{Name: "x"}
	if r.Toolkit()["a"].Name == "x" {
		t.Fatalf("toolkit was modified externally")
	}
}

func TestRuntimeQueueEvent(t *testing.T) {
	r := NewRuntime(nil, nil, nil, nil, false)
	r.QueueEvent(Event{})
	if len(r.events) != 1 {
		t.Fatalf("unexpected event queue length: %d", len(r.events))
	}
}
