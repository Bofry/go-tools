package main

const (
	BOOL_ASSERTION_TYPE        = "bool"
	STRING_ASSERTION_TYPE      = "string"
	STRING_PTR_ASSERTION_TYPE  = "*string"
	INT_ASSERTION_TYPE         = "int"
	INT_PTR_ASSERTION_TYPE     = "*int"
	FLOAT_ASSERTION_TYPE       = "float"
	FLOAT_PTR_ASSERTION_TYPE   = "*float"
	NUMBER_ASSERTION_TYPE      = "number"
	NUMBER_PTR_ASSERTION_TYPE  = "*number"
	DECIMAL_ASSERTION_TYPE     = "decimal"
	DECIMAL_PTR_ASSERTION_TYPE = "*decimal"
	VALUE_ASSERTION_TYPE       = "value"
	IP_ASSERTION_TYPE          = "ip"
)

type (
	ImportDirective struct {
		Name string
		Path string
	}

	AssertorFile struct {
		PackageName string
		Imports     []*ImportDirective
		Types       []*AssertorType
	}

	AssertorType struct {
		Name           string
		SourceTypeName string
		Assertions     []*AssertorValueAssertion
	}

	AssertorValueAssertion struct {
		TypeName          string
		Name              string
		Tag               string
		Type              string
		ArgvFieldType     string
		ArgvFieldTypeStar string // e.g: "", "*" or "**" ...
		Skipped           bool
	}
)
