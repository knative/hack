package unit_test

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

var (
	//go:embed scripts/*.bash
	scripts      embed.FS
	bashQuotesRx = regexp.MustCompile("(?m)^#.*")
	toMuchNlRx   = regexp.MustCompile("(?m)\n{3,}")
)

func aborted(msg string) []check {
	fmsg := fmt.Sprintf("ERROR: %s", msg)
	return equal(makeBanner('*', fmsg))
}

func makeBanner(ch rune, msg string) string {
	const span = 4
	border := strings.Repeat(string(ch), len(msg)+span*2+2)
	side := strings.Repeat(string(ch), span)
	return strings.Join([]string{
		border,
		side + " " + msg + " " + side,
		border,
		side + " 2018-07-18 23:00:00.000000000+00:00",
		border,
	}, "\n") + "\n"
}

func empty() []check {
	return equal("")
}

func lines(strs ...string) []check {
	return equal(strings.Join(strs, "\n") + "\n")
}

func contains(str string) check {
	return func(t assert.TestingT, output string) {
		assert.Contains(t, output, str)
	}
}

func equal(str string) []check {
	return []check{func(t assert.TestingT, output string) {
		assert.Equal(t, str, output)
	}}
}

type check func(t assert.TestingT, output string)

type testCase struct {
	name     string
	commands []string
	retcode  *returnCode
	stdout   []check
	stderr   []check
}

type returnCode int

func retcode(code int) *returnCode {
	rc := returnCode(code)
	return &rc
}

func (tc testCase) test(sc shellScript) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		code, out, err, src := sc.run(t, tc.testCommands())
		tc.validRetcode(t, code)
		for _, chck := range coalesce(tc.stdout, equal("")) {
			chck(t, out)
		}
		for _, chck := range coalesce(tc.stderr, equal("")) {
			chck(t, err)
		}
		if t.Failed() {
			t.Logf("Retcode: %v", code)
			t.Logf("Stdout: \n---\n%v---\n", out)
			t.Logf("Stderr: \n---\n%v---\n", err)
			t.Logf("Shell script source: \n---\n%v---\n", src)
		}
	}
}

func coalesce(chcks ...[]check) []check {
	for _, chck := range chcks {
		if len(chck) > 0 {
			return chck
		}
	}
	return []check{}
}

func (tc testCase) testCommands() []string {
	if len(tc.commands) > 0 {
		return tc.commands
	}
	return []string{tc.name}
}

func (tc testCase) validRetcode(t TestingT, gotRetcode int) {
	if tc.retcode != nil {
		assert.Equal(t, int(*tc.retcode), gotRetcode)
		return
	}
	if len(tc.stderr) > 0 {
		assert.NotEqual(t, 0, gotRetcode)
	} else {
		assert.Equal(t, 0, gotRetcode)
	}
}

type scriptlet func(t TestingT) string

func newShellScript(scriptlets ...scriptlet) shellScript {
	return shellScript{
		append(scriptlets, mockBinary("date", map[string]string{
			"": "2018-07-18 23:00:00.000000000+00:00",
		})),
	}
}

type shellScript struct {
	scriptlets []scriptlet
}

func loadFile(names ...string) scriptlet {
	return func(t TestingT) string {
		sc := make([]scriptlet, 0, len(names))
		for i := range names {
			name := names[i]
			sc = append(sc, func(t TestingT) string {
				byts, err := scripts.ReadFile(path.Join("scripts", name))
				require.NoError(t, err)
				return string(byts)
			})
		}
		src := make([]string, len(sc))
		for i, s := range sc {
			src[i] = s(t)
		}
		return strings.Join(src, "\n")
	}
}

func instructions(inst ...string) scriptlet {
	return func(t TestingT) string {
		return strings.Join(inst, "\n")
	}
}

func mockBinary(name string, responses map[string]string) scriptlet {
	return func(t TestingT) string {
		code := make([]string, 0, len(responses)*10)
		code = append(code,
			fmt.Sprintf(`cat > "${TMPPATH}/%s" <<'EOF'`, name),
			"#!/usr/bin/env bash")
		for args, response := range responses {
			code = append(code, fmt.Sprintf(`if [[ "$*" == *"%s"* ]]; then`, args))
			for _, li := range strings.Split(response, "\n") {
				code = append(code, fmt.Sprintf(`  echo "%s"`, li))
			}
			code = append(code,
				"  exit 0",
				"fi")
		}
		code = append(code,
			fmt.Sprintf(`echo "%s $*"`, name),
			"EOF",
			fmt.Sprintf(`chmod +x "${TMPPATH}/%s"`, name),
		)
		return strings.Join(code, "\n") + "\n"
	}
}

type pair struct {
	key, val string
}

func mockGo(pairs ...pair) scriptlet {
	vals := map[string]string{}
	for _, p := range pairs {
		vals[p.key] = p.val
	}
	return mockBinary("go", vals)
}

func mockKubectl(responses map[string]string) scriptlet {
	return mockBinary("kubectl", responses)
}

func mockGcloud() scriptlet {
	return mockBinary("gcloud", map[string]string{})
}

type TestingT interface {
	TempDir() string
	Logf(format string, args ...interface{})
	require.TestingT
}

func (s shellScript) run(t TestingT, commands []string) (int, string, string, string) {
	src := s.source(t, commands)
	sf := s.write(t, src)
	defer func(name string) {
		require.NoError(t, os.Remove(name))
	}(sf)
	rootDir := path.Dir(path.Dir(path.Dir(sf)))
	c := exec.Command("bash", strings.ReplaceAll(sf, rootDir, "."))
	var bo, be bytes.Buffer
	c.Stdout = &bo
	c.Stderr = &be
	c.Dir = rootDir
	err := c.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			require.NoError(t, err)
		}
	}
	return c.ProcessState.ExitCode(), bo.String(), be.String(), src
}

func (s shellScript) source(t TestingT, commands []string) string {
	source := fmt.Sprintf(`
set -Eeuo pipefail
export TMPPATH='%s'
export PATH="${TMPPATH}:${PATH}"
`, t.TempDir())
	bashShebang := "#!/usr/bin/env bash\n"
	for _, sclet := range s.scriptlets {
		source += strings.TrimPrefix(sclet(t), bashShebang)
	}
	source = bashShebang + "\n" +
		bashQuotesRx.ReplaceAllStringFunc(source, func(in string) string {
			if strings.HasPrefix(in, "#!/") {
				return in
			}
			return ""
		}) + "\n"
	for _, command := range commands {
		source += command + "\n"
	}

	source = toMuchNlRx.ReplaceAllString(source, "\n\n")
	return source
}

func (s shellScript) write(t TestingT, src string) string {
	dir := currentDir()
	p := path.Join(dir, fmt.Sprintf("unittest-%s.bash", rand.String(12)))
	err := os.WriteFile(p, []byte(src), 0o600)
	require.NoError(t, err)
	return p
}

func currentDir() string {
	_, file, _, _ := runtime.Caller(0)
	return path.Dir(file)
}
