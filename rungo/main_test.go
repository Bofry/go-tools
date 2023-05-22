package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const (
	FILE_GO_MOD         = "go.mod"
	FILE_GO_MOD_CONTENT = `module rungo-demo

go 1.20
`

	FILE_ENV         = ".env"
	FILE_ENV_CONTENT = `Environment=local
EnvFoo=bar
`
	FILE_ENV_TEST         = ".env.test"
	FILE_ENV_TEST_CONTENT = `Environment=test
EnvFoo=foobar
`
	FILE_APP_GO         = "app.go"
	FILE_APP_GO_CONTENT = `package main

import (
	"fmt"
	"os"
)

func main() {
	for i := 1; i < len(os.Args); i++ {
		v := os.Args[i]
		fmt.Printf("ARG[%d]: %v\n", i, v)
	}
	for _, k := range []string{"Environment", "EnvFoo"} {
		v := os.Getenv(k)
		fmt.Printf("ENV[%s]: %s\n", k, v)
	}
	fmt.Println("Hello, World")
}
`
)

var (
	globalTempDir     string
	globalTempDirOnce sync.Once
)

func Test_WithEnvTest(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})
	t.Logf("%v", tmp)

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	assert(t,
		createTempFiles(tmp, FILE_GO_MOD_CONTENT, FILE_GO_MOD),
		createTempFiles(tmp, FILE_ENV_CONTENT, FILE_ENV),
		createTempFiles(tmp, FILE_ENV_TEST_CONTENT, FILE_ENV_TEST),
		createTempFiles(tmp, FILE_APP_GO_CONTENT, FILE_APP_GO),
	)

	defaultStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Chdir(tmp)
	os.Args = []string{
		"rungo",
		"-f",
		`.env.test`,
		"app.go",
		"-foo",
		"bar",
	}
	// NOTE: avoid painc when call os.Exit() under testing
	osExit = func(i int) {} // do nothing
	main()
	os.Chdir(workdir)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = defaultStdout

	{
		expectedContent := []byte(strings.Join([]string{
			`ARG[1]: -foo`,
			`ARG[2]: bar`,
			`ENV[Environment]: test`,
			`ENV[EnvFoo]: foobar`,
			`Hello, World`,
			"",
		}, "\n"))
		if !reflect.DeepEqual(expectedContent, out) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", string(expectedContent), string(out))
		}
	}

	// Output:
	// ARG[1]: -foo
	// ARG[2]: bar
	// ENV[Environment]: test
	// ENV[EnvFoo]: foobar
	// Hello, World
}

func Test_WithDefaultEnv(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})
	t.Logf("%v", tmp)

	workdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	assert(t,
		createTempFiles(tmp, FILE_GO_MOD_CONTENT, FILE_GO_MOD),
		createTempFiles(tmp, FILE_ENV_CONTENT, FILE_ENV),
		createTempFiles(tmp, FILE_ENV_TEST_CONTENT, FILE_ENV_TEST),
		createTempFiles(tmp, FILE_APP_GO_CONTENT, FILE_APP_GO),
	)

	defaultStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Chdir(tmp)
	os.Args = []string{
		"rungo",
		"app.go",
		"-foo",
		"bar",
	}
	// NOTE: avoid painc when call os.Exit() under testing
	osExit = func(i int) {} // do nothing
	main()
	os.Chdir(workdir)

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = defaultStdout

	{
		expectedContent := []byte(strings.Join([]string{
			`ARG[1]: -foo`,
			`ARG[2]: bar`,
			`ENV[Environment]: local`,
			`ENV[EnvFoo]: bar`,
			`Hello, World`,
			"",
		}, "\n"))
		if !reflect.DeepEqual(expectedContent, out) {
			t.Errorf("app.go expect:\n%s\ngot:\n%s\n", string(expectedContent), string(out))
		}
	}

	// Output:
	// ARG[1]: -foo
	// ARG[2]: bar
	// ENV[Environment]: local
	// ENV[EnvFoo]: bar
	// Hello, World
}

func assert(t *testing.T, err ...error) {
	for _, e := range err {
		if e != nil {
			t.Fatal(e)
		}
	}
}

func getTmpDir(t *testing.T) string {
	globalTempDirOnce.Do(func() {
		globalTempDir = t.TempDir()
	})
	return globalTempDir
}

func createTempFiles(tmpPath string, content string, filename string) error {
	fs, err := os.Create(path.Join(tmpPath, filename))
	defer fs.Close()
	if err != nil {
		return fmt.Errorf("cannot create file '%s' cause %v", filename, err)
	}
	_, err = fmt.Fprint(fs, content)
	return err
}
