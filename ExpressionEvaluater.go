package main

import (
	"fmt"
	"github.com/Knetic/govaluate"
	"strconv"
)

type ConditionObject struct {
	ConditionType            string `json:"conditionType"`       //TRIGGER, SKIP, SUCCESS, FAILURE
	ConditionOnVariable      string `json:"conditionOnVariable"` //name of variable
	ConditionalOperator      string `json:"conditionalOperator"`
	ConditionalValue         string `json:"conditionalValue"`
	typecastConditionalValue interface{}
}

func evaluateExpression(condition ConditionObject, variables []*VariableObject) (status bool, err error) {
	variableMap := make(map[string]*VariableObject)
	for _, variable := range variables {
		variableMap[variable.Name] = variable
	}
	variableOperand := variableMap[condition.ConditionOnVariable]
	if variableOperand.DeducedValue == nil {
		converted, err := typeConverter(variableOperand.Value, variableOperand.Format)
		if err != nil {
			return false, err
		}
		variableOperand.DeducedValue = converted
	}
	refOperand, err := typeConverter(condition.ConditionalValue, variableOperand.Format)
	if err != nil {
		return false, err
	}
	expression, err := govaluate.NewEvaluableExpression(fmt.Sprintf("variableOperand %s refOperand", condition.ConditionalOperator))
	if err != nil {
		return false, err
	}
	parameters := make(map[string]interface{}, 8)
	parameters["variableOperand"] = variableOperand.DeducedValue
	parameters["refOperand"] = refOperand
	result, err := expression.Evaluate(parameters)
	if err != nil {
		return false, err
	}
	status = result.(bool)
	return status, nil
}

func typeConverter(value string, format Format) (interface{}, error) {
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

type Format int

const (
	STRING Format = iota
	NUMBER
	BOOL
	DATE
)

func (d Format) ValuesOf(format string) Format {
	if format == "NUMBER" {
		return NUMBER
	} else if format == "BOOL" {
		return BOOL
	} else if format == "STRING" {
		return STRING
	} else if format == "DATE" {
		return DATE
	}
	return STRING
}

func (d Format) String() string {
	return [...]string{"NUMBER", "BOOL", "STRING", "DATE"}[d]
}
