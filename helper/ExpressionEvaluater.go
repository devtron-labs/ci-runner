package helper

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"strconv"
)

type ConditionObject struct {
	ConditionType            ConditionType `json:"conditionType"`       //TRIGGER, SKIP, SUCCESS, FAIL
	ConditionOnVariable      string        `json:"conditionOnVariable"` //name of variable
	ConditionalOperator      string        `json:"conditionalOperator"`
	ConditionalValue         string        `json:"conditionalValue"`
	typecastConditionalValue interface{}
}

func ShouldTriggerStage(conditions []*ConditionObject, variables []*VariableObject) (bool, error) {
	conditionType := conditions[0].ConditionType //assuming list has min 1
	status := true
	for _, condition := range conditions {
		result, err := evaluateExpression(condition, variables)
		if err != nil {
			return false, err
		}
		status = status && result
	}
	if conditionType == TRIGGER {
		return status, nil // trigger if all success
	} else {
		return !status, nil //skip if all ture
	}
}

func StageIsSuccess(conditions []*ConditionObject, variables []*VariableObject) (bool, error) {
	conditionType := conditions[0].ConditionType //assuming list has min 1
	status := true
	for _, condition := range conditions {
		result, err := evaluateExpression(condition, variables)
		if err != nil {
			return false, err
		}
		status = status && result
	}
	if conditionType == SUCCESS {
		return status, nil // success if all success
	} else {
		return !status, nil //fail if all success
	}
}

func evaluateExpression(condition *ConditionObject, variables []*VariableObject) (status bool, err error) {
	variableMap := make(map[string]*VariableObject)
	for _, variable := range variables {
		variableMap[variable.Name] = variable
	}
	variableOperand := variableMap[condition.ConditionOnVariable]
	if variableOperand.TypedValue == nil {
		converted, err := TypeConverter(variableOperand.Value, variableOperand.Format)
		if err != nil {
			return false, err
		}
		variableOperand.TypedValue = converted
	}
	refOperand, err := TypeConverter(condition.ConditionalValue, variableOperand.Format)
	if err != nil {
		return false, err
	}
	expression, err := govaluate.NewEvaluableExpression(fmt.Sprintf("variableOperand %s refOperand", condition.ConditionalOperator))
	if err != nil {
		return false, err
	}
	parameters := make(map[string]interface{}, 8)
	parameters["variableOperand"] = variableOperand.TypedValue
	parameters["refOperand"] = refOperand
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return false, err
	}
	status = result.(bool)
	return status, nil
}

func TypeConverter(value string, format Format) (interface{}, error) {
	switch format {
	case STRING:
		return value, nil
	case NUMBER:
		return strconv.ParseFloat(value, 8)
	case BOOL:
		return strconv.ParseBool(value)
	case DATE:
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported datatype")
	}
}
