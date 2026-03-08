package chat

import "testing"

func TestNewSearchQualifier_CustomModel(t *testing.T) {
	sq := NewSearchQualifier(nil, "custom-model:7b")
	if sq.model != "custom-model:7b" {
		t.Errorf("got model %q, want %q", sq.model, "custom-model:7b")
	}
}

func TestNewSearchQualifier_DefaultModel(t *testing.T) {
	sq := NewSearchQualifier(nil, "")
	if sq.model != defaultQualifierModel {
		t.Errorf("got model %q, want default %q", sq.model, defaultQualifierModel)
	}
}
