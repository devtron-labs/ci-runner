package main

import "testing"

func Test_evaluateExpression(t *testing.T) {
	type args struct {
		condition ConditionObject
		variables map[string]interface{}
	}
	tests := []struct {
		name       string
		args       args
		wantStatus bool
		wantErr    bool
	}{
		/*{name: "Eval_false",
		args: args{condition: ConditionObject{
			//ConditionType:       "",
			ConditionOnVariable: "age",
			ConditionalOperator: ">",

			ConditionalValue: "10",
		}, variables: map[string]interface{}{"age": 8}},
		wantErr:    false,
		wantStatus: false},*/
		/*		{name: "Eval_true_date",
				args: args{condition: ConditionObject{
					//ConditionType:       "",
					ConditionOnVariable: "today",
					ConditionalOperator: ">",
					ConditionalValue:    "Tue Apr 10 13:55:21 IST 2022",
				}, variables: map[string]interface{}{"today": "Tue Apr 12 13:55:21 IST 2022"}},
				wantErr:    false,
				wantStatus: true},*/
		{name: "Eval_false_date",
			args: args{condition: ConditionObject{
				//ConditionType:       "",
				ConditionOnVariable: "today",
				ConditionalOperator: ">",
				ConditionalValue:    "'Tue Apr 10 13:55:21 IST 2022'",
			}, variables: map[string]interface{}{"today": "'Tue Apr 8 13:55:21 IST 2022'"}},
			wantErr:    false,
			wantStatus: false},
		/*{name: "eval_true",
		args: args{condition: ConditionObject{
			//ConditionType:       "",
			ConditionOnVariable: "age",
			ConditionalOperator: ">",
			ConditionalValue:    "10",
		}, variables: map[string]string{"age": "12"}},
		wantErr:    false,
		wantStatus: true},*/
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
