package policy

import "testing"

func TestPlanMode(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    string
		wantErr bool
	}{
		{name: "defaults to auto, so existing files behave as before", yaml: "feature: p\n", want: "auto"},
		{name: "hold", yaml: "stages:\n  plan: hold\n", want: "hold"},
		{name: "conditional", yaml: "stages:\n  plan: conditional\n", want: "conditional"},
		{name: "an unknown mode is refused", yaml: "stages:\n  plan: sometimes\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := Parse([]byte(tt.yaml))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := file.PlanMode(); got != tt.want {
				t.Fatalf("PlanMode = %q, want %q", got, tt.want)
			}
		})
	}
}
