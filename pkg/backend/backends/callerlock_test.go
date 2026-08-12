package backends

import (
	"reflect"
	"testing"
)

func TestBuildCallerRules(t *testing.T) {
	got := BuildCallerRules(CallerSSHPorts(), []string{"1.2.3.4/32", "10.0.0.0/8"})
	want := []CallerRule{
		{Protocol: ProtocolTCP, FromPort: 22, ToPort: 22, Cidr: "1.2.3.4/32"},
		{Protocol: ProtocolTCP, FromPort: 22, ToPort: 22, Cidr: "10.0.0.0/8"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BuildCallerRules() = %v, want %v", got, want)
	}
}

func TestBuildCallerRulesDeduplicates(t *testing.T) {
	got := BuildCallerRules(CallerSSHPorts(), []string{"1.2.3.4/32", "1.2.3.4/32"})
	if len(got) != 1 {
		t.Errorf("BuildCallerRules() = %v, want a single rule", got)
	}
}

func TestDiffCallerRules(t *testing.T) {
	ssh := func(cidr string) CallerRule {
		return CallerRule{Protocol: ProtocolTCP, FromPort: 22, ToPort: 22, Cidr: cidr}
	}
	tests := []struct {
		name       string
		existing   []CallerRule
		desired    []CallerRule
		wantAdd    []CallerRule
		wantRemove []CallerRule
	}{
		{
			name:       "empty group gets the rule",
			existing:   nil,
			desired:    []CallerRule{ssh("1.2.3.4/32")},
			wantAdd:    []CallerRule{ssh("1.2.3.4/32")},
			wantRemove: []CallerRule{},
		},
		{
			name:       "already correct is a no-op",
			existing:   []CallerRule{ssh("1.2.3.4/32")},
			desired:    []CallerRule{ssh("1.2.3.4/32")},
			wantAdd:    []CallerRule{},
			wantRemove: []CallerRule{},
		},
		{
			name:       "caller IP moved",
			existing:   []CallerRule{ssh("1.2.3.4/32")},
			desired:    []CallerRule{ssh("5.6.7.8/32")},
			wantAdd:    []CallerRule{ssh("5.6.7.8/32")},
			wantRemove: []CallerRule{ssh("1.2.3.4/32")},
		},
		{
			name:       "widened to several CIDRs",
			existing:   []CallerRule{ssh("1.2.3.4/32")},
			desired:    []CallerRule{ssh("1.2.3.4/32"), ssh("10.0.0.0/8")},
			wantAdd:    []CallerRule{ssh("10.0.0.0/8")},
			wantRemove: []CallerRule{},
		},
		{
			name:     "port range change is not confused with a CIDR change",
			existing: []CallerRule{ssh("1.2.3.4/32")},
			desired: []CallerRule{
				{Protocol: ProtocolTCP, FromPort: 3000, ToPort: 3010, Cidr: "1.2.3.4/32"},
			},
			wantAdd: []CallerRule{
				{Protocol: ProtocolTCP, FromPort: 3000, ToPort: 3010, Cidr: "1.2.3.4/32"},
			},
			wantRemove: []CallerRule{ssh("1.2.3.4/32")},
		},
		{
			name:       "nothing desired revokes everything we own",
			existing:   []CallerRule{ssh("1.2.3.4/32"), ssh("5.6.7.8/32")},
			desired:    []CallerRule{},
			wantAdd:    []CallerRule{},
			wantRemove: []CallerRule{ssh("1.2.3.4/32"), ssh("5.6.7.8/32")},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			add, remove := DiffCallerRules(test.existing, test.desired)
			if !reflect.DeepEqual(add, test.wantAdd) {
				t.Errorf("add = %v, want %v", add, test.wantAdd)
			}
			if !reflect.DeepEqual(remove, test.wantRemove) {
				t.Errorf("remove = %v, want %v", remove, test.wantRemove)
			}
		})
	}
}
