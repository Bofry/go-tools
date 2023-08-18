package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

var (
	_FILE_ARGV_GO = strings.ReplaceAll(`package demo

import (
	"github.com/Bofry/arg"
)

type State struct {
	Remark string
}

//go:generate gen-bofry-arg-assertor
type ProtagonistArgv struct /* tag=json */ {
	ID        string      ”json:"id"    arg:"_id"  ^:"arg"”
	Name      string      ”json:"name"”
	Age       int         ”json:"age"”
	Status    *string     ”json:"status"”
	IP        *arg.IP     ”json:"ip"”
	Timestamp *Timestamp  ”json:"timestamp"”
	State     *State      ”json:"state"”
	Token     *arg.Number ”json:"token"”
	Bounty    float64     ”json:"bounty"”
	OnStage   bool        ”json:"onstage"”
}
`, "”", "`")

	_FILE_TIMESTAMP_GO = `package demo

type Timestamp int
`

	_EXPECT_FILE_ARGV_ASSERTOR_GO = `package demo

import (
	arg "github.com/Bofry/arg"
)

type ProtagonistArgvAssertor struct {
	argv *ProtagonistArgv
}

func (argv *ProtagonistArgv) Assertor() *ProtagonistArgvAssertor {
	return &ProtagonistArgvAssertor{
		argv: argv,
	}
}

func (assertor *ProtagonistArgvAssertor) ID(validators ...arg.StringValidator) error {
	return arg.Strings.Assert(assertor.argv.ID, "_id",
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

func (assertor *ProtagonistArgvAssertor) Status(validators ...arg.StringPtrValidator) error {
	return arg.StringPtr.Assert(assertor.argv.Status, "status",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) IP(validators ...arg.IPValidator) error {
	return arg.IPs.Assert(*assertor.argv.IP, "ip",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Timestamp(validators ...arg.IntPtrValidator) error {
	var v *int64 = nil
	if assertor.argv.Timestamp != nil {
		var scalar = int64(*assertor.argv.Timestamp)
		v = &scalar
	}
	return arg.IntPtr.Assert(v, "timestamp",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) State(validators ...arg.ValueValidator) error {
	return arg.Values.Assert(assertor.argv.State, "state",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Token(validators ...arg.NumberPtrValidator) error {
	return arg.NumberPtr.Assert(assertor.argv.Token, "token",
		validators...,
	)
}

func (assertor *ProtagonistArgvAssertor) Bounty(validators ...arg.FloatValidator) error {
	return arg.Floats.Assert(assertor.argv.Bounty, "bounty",
		validators...,
	)
}
`
)

func Test(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(tmp)
		os.Clearenv()
	})

	assert(t,
		createTempFiles(tmp, _FILE_ARGV_GO, "protagonistArgv.go", ""),
		createTempFiles(tmp, _FILE_TIMESTAMP_GO, "timestamp.go", ""),
	)

	os.Args = []string{
		"gen-bofry-arg-assertor",
		"-target",
		path.Join(tmp, "protagonistArgv.go"),
	}
	// NOTE: avoid painc when call os.Exit() under testing
	osExit = func(i int) {
		if i != 0 {
			t.Fatalf("got exit code %d", i)
		}
	}
	main()

	{
		content, err := readFile(tmp, "protagonistArgvAssertor_gen.go", "")
		if err != nil {
			t.Fatal(err)
		}
		expectedContent := _EXPECT_FILE_ARGV_ASSERTOR_GO
		if expectedContent != string(content) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", expectedContent, string(content))
		}
	}

}

func assert(t *testing.T, err ...error) {
	for _, e := range err {
		if e != nil {
			t.Fatal(e)
		}
	}
}

func createTempFiles(tmpPath string, content string, filename string, subDir string) error {
	if len(subDir) > 0 {
		tmpPath = path.Join(tmpPath, subDir)

		_, err := os.Stat(tmpPath)
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot create dir '%s' cause %v", subDir, err)
		}
		err = os.Mkdir(tmpPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("cannot create dir '%s' cause %v", subDir, err)
		}
	}
	fs, err := os.Create(path.Join(tmpPath, filename))
	defer fs.Close()
	if err != nil {
		return fmt.Errorf("cannot create file '%s' cause %v", filename, err)
	}
	_, err = fmt.Fprint(fs, content)
	return err
}

func readFile(tmpPath string, filename string, subDir string) ([]byte, error) {
	if len(subDir) > 0 {
		tmpPath = path.Join(tmpPath, subDir)
	}

	filepath := path.Join(tmpPath, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("cannot open dir '%s' cause %v", filename, err)
	}
	return content, nil
}
