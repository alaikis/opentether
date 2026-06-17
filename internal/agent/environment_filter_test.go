package agent

import "testing"

func TestStripEnvironmentDetailsFromPrompt(t *testing.T) {
	input := "查询订单\n<environment_details>\nCurrent time: x\nWorking directory: y\n</environment_details>"
	got := stripEnvironmentDetailsFromPrompt(input)
	if got != "查询订单" {
		t.Fatalf("unexpected stripped prompt: %q", got)
	}
}
