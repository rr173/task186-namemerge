package view

import "testing"

func TestTask186Bug08_EmptyViewEvaluationIsSafe(t *testing.T) {
	ev := &Evaluator{}
	result, err := ev.Evaluate("empty-view")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Clusters) != 0 || len(result.Roles) != 0 || len(result.Conflicts) != 0 {
		t.Fatalf("empty evaluation = %+v, want no results", result)
	}
}
