package main

import "testing"

func Test_evaluateExpression(t *testing.T) {
	type args struct {
		condition *ConditionObject
		variables []*VariableObject
	}
	tests := []struct {
		name       string
		args       args
		wantStatus bool
		wantErr    bool
	}{
		{name: "Eval_false",
			args: args{condition: &ConditionObject{
				ConditionOnVariable: "age",
				ConditionalOperator: ">",
				ConditionalValue:    "10",
			}, variables: []*VariableObject{{Name: "age", Value: "8", Format: NUMBER}}},
			wantErr:    false,
			wantStatus: false},
		{name: "eval_true",
			args: args{condition: &ConditionObject{
				//ConditionType:       "",
				ConditionOnVariable: "age",
				ConditionalOperator: ">",
				ConditionalValue:    "10",
			}, variables: []*VariableObject{{Name: "age", Value: "12", Format: NUMBER}}},
			wantErr:    false,
			wantStatus: true},
		{name: "Eval_true_date",
			args: args{condition: &ConditionObject{
				ConditionOnVariable: "today",
				ConditionalOperator: ">",
				ConditionalValue:    "Tue Apr 10 13:55:21 IST 2022",
			}, variables: []*VariableObject{{Name: "today", Value: "Tue Apr 12 13:55:21 IST 2022", Format: DATE}}},
			wantErr:    false,
			wantStatus: true},
		{name: "Eval_false_date",
			args: args{condition: &ConditionObject{
				//ConditionType:       "",
				ConditionOnVariable: "today",
				ConditionalOperator: "<",
				ConditionalValue:    "'Tue Apr 10 13:55:21 IST 2022'",
			}, variables: []*VariableObject{{Name: "today", Value: "'Tue Apr 8 13:55:21 IST 2022'", Format: DATE}}},
			wantErr:    false,
			wantStatus: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, err := evaluateExpression(tt.args.condition, tt.args.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("evaluateExpression() gotStatus = %v, want %v", gotStatus, tt.wantStatus)
			}
		})
	}
}
