package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAssertorFileWriter_Write(t *testing.T) {
	file := &AssertorFile{
		PackageName: "test",
		Imports: []*ImportDirective{
			&ImportDirective{
				Path: "net",
			},
		},
		Types: []*AssertorType{
			&AssertorType{
				Name:           "ProtagonistArgvAssertor",
				SourceTypeName: "ProtagonistArgv",
				Assertions: []*AssertorValueAssertion{
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "ID",
						Tag:               "id",
						Type:              "string",
						ArgvFieldType:     "string",
						ArgvFieldTypeStar: "",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "Name",
						Tag:               "name",
						Type:              "string",
						ArgvFieldType:     "string",
						ArgvFieldTypeStar: "",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "Age",
						Tag:               "age",
						Type:              "int",
						ArgvFieldType:     "int",
						ArgvFieldTypeStar: "",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "Status",
						Tag:               "status",
						Type:              "string",
						ArgvFieldType:     "string",
						ArgvFieldTypeStar: "*",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "IP",
						Tag:               "ip",
						Type:              "ip",
						ArgvFieldType:     "net.IP",
						ArgvFieldTypeStar: "*",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "Timestamp",
						Tag:               "timestamp",
						Type:              "int",
						ArgvFieldType:     "*Timestamp",
						ArgvFieldTypeStar: "*",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "State",
						Tag:               "state",
						Type:              "value",
						ArgvFieldType:     "State",
						ArgvFieldTypeStar: "*",
					},
					&AssertorValueAssertion{
						TypeName:          "ProtagonistArgvAssertor",
						Name:              "Token",
						Tag:               "token",
						Type:              "number",
						ArgvFieldType:     "arg.Number",
						ArgvFieldTypeStar: "*",
					},
				},
			},
		},
	}

	var buf bytes.Buffer

	writer := NewAssertorFileWriter()
	writer.Write(&buf, file)

	expectedOutput := []byte(`package test

import (
	arg "github.com/Bofry/arg"
	"net"
)

type ProtagonistArgvAssertor struct {
	argv  *ProtagonistArgv
}

func (argv *ProtagonistArgv) Assertor() *ProtagonistArgvAssertor {
	return &ProtagonistArgvAssertor{
		argv: argv,
	}
}

func (assertor *ProtagonistArgvAssertor) ID(validators ...arg.StringValidator) error {
	return arg.Strings.Assert(assertor.argv.ID, "id",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Name(validators ...arg.StringValidator) error {
	return arg.Strings.Assert(assertor.argv.Name, "name",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Age(validators ...arg.IntValidator) error {
	return arg.Ints.Assert(int64(assertor.argv.Age), "age",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Status(validators ...arg.StringValidator) error {
	return arg.Strings.Assert(*assertor.argv.Status, "status",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) IP(validators ...arg.IPValidator) error {
	return arg.IPs.Assert(*assertor.argv.IP, "ip",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Timestamp(validators ...arg.IntValidator) error {
	return arg.Ints.Assert(int64(*assertor.argv.Timestamp), "timestamp",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) State(validators ...arg.ValueValidator) error {
	return arg.Values.Assert(*assertor.argv.State, "state",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Token(validators ...arg.NumberValidator) error {
	return arg.Numbers.Assert(*assertor.argv.Token, "token",
		validators...,
	)
}
`)

	if !reflect.DeepEqual(expectedOutput, buf.Bytes()) {
		t.Errorf("unit test failed. output doesn't match expected.")
	}
}
