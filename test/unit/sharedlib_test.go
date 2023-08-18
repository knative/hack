package unit_test

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func warned(msg string) []check {
	fmsg := fmt.Sprintf("WARN: %s", msg)
	return equal(makeBanner('!', fmsg))
}

func makeBanner(ch rune, msg string) string {
	const span = 4
	border := strings.Repeat(string(ch), len(msg)+span*2+2)
	side := strings.Repeat(string(ch), span)
	return strings.Join([]string{
		border,
		side + " " + msg + " " + side,
		border,
		side + " 2018-07-18 23:00:00",
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

type scriptlet interface {
	scriptlet(t TestingT) string
}

type fnScriptlet func(t TestingT) string

func (f fnScriptlet) scriptlet(t TestingT) string {
	return f(t)
}

func newShellScript(scriptlets ...scriptlet) shellScript {
	return shellScript{
		append(scriptlets, mockBinary("date", response{
			anyArgs{}, simply("2018-07-18 23:00:00"),
		})),
	}
}

type shellScript struct {
	scriptlets []scriptlet
}

func loadFile(names ...string) scriptlet {
	return fnScriptlet(func(t TestingT) string {
		sc := make([]scriptlet, 0, len(names))
		for i := range names {
			name := names[i]
			sc = append(sc, fnScriptlet(func(t TestingT) string {
				byts, err := scripts.ReadFile(path.Join("scripts", name))
				require.NoError(t, err)
				return string(byts)
			}))
		}
		src := make([]string, len(sc))
		for i, s := range sc {
			src[i] = s.scriptlet(t)
		}
		return strings.Join(src, "\n")
	})
}

func envs(envs map[string]string) scriptlet {
	instr := make([]string, 0, len(envs))
	for k, v := range envs {
		instr = append(instr, fmt.Sprintf(`export %s="%s"`, k, v))
	}
	return instructions(instr...)
}

func instructions(inst ...string) scriptlet {
	return fnScriptlet(func(t TestingT) string {
		return strings.Join(inst, "\n")
	})
}

type simply string

func (s simply) Invocations(_ string) []string {
	ls := strings.Split(string(s), "\n")
	code := make([]string, len(ls))
	for i, li := range ls {
		code[i] = fmt.Sprintf(`  echo "%s"`, li)
	}
	return code
}

type callOriginal struct{}

func (o callOriginal) Invocations(bin string) []string {
	binPath, err := exec.LookPath(bin)
	if err != nil {
		panic(err)
	}
	return []string{
		fmt.Sprintf(`  '%s' "$@"`, binPath),
	}
}

func mockBinary(name string, responses ...response) scriptlet {
	return fnScriptlet(func(t TestingT) string {
		code := make([]string, 0, len(responses)*10)
		code = append(code,
			fmt.Sprintf(`cat > "${TMPPATH}/%s" <<'EOF'`, name),
			"#!/usr/bin/env bash")
		for _, p := range responses {
			code = append(code, fmt.Sprintf(`if [[ "$*" == %s ]]; then`, p.args))
			code = append(code, p.response.Invocations(name)...)
			code = append(code,
				"  exit $?",
				"fi")
		}
		code = append(code,
			fmt.Sprintf(`echo "%s $*"`, name),
			"EOF",
			fmt.Sprintf(`chmod +x "${TMPPATH}/%s"`, name),
		)
		return strings.Join(code, "\n") + "\n"
	})
}

type invocations interface {
	Invocations(bin string) []string
}

type args interface {
	fmt.Stringer
}

type response struct {
	args
	response invocations
}

type startsWith struct {
	prefix string
}

func (s startsWith) String() string {
	return fmt.Sprintf(`"%s"*`, s.prefix)
}

type anyArgs struct{}

func (a anyArgs) String() string {
	return "*"
}

func mockGo(responses ...response) scriptlet {
	lstags := "knative.dev/toolbox/go-ls-tags@latest"
	modscope := "knative.dev/toolbox/modscope@latest"
	callOriginals := []args{
		startsWith{"run " + lstags},
		startsWith{"run " + modscope},
		startsWith{"list"},
		startsWith{"env"},
		startsWith{"version"},
	}
	originalResponses := make([]response, len(callOriginals))
	for i, co := range callOriginals {
		originalResponses[i] = response{co, callOriginal{}}
	}
	return prefetchScriptlet{
		delegate: mockBinary("go", append(originalResponses, responses...)...),
		prefetchers: []prefetcher{
			goRunHelpPrefetcher(lstags),
			goRunHelpPrefetcher(modscope),
		},
	}
}

func mockKubectl(responses ...response) scriptlet {
	return mockBinary("kubectl", append([]response{{
		startsWith{"config current-context"}, simply("gke_deadbeef_1.24"),
	}, {
		startsWith{"get pods --no-headers -n"},
		simply("beef-e3c1 1/1 Running 0 2s\nceed-45b3 1/1 Running 0 1s"),
	}}, responses...)...)
}

func fakeProwJob() scriptlet {
	return union(
		loadFile("fake-prow-job.bash"),
		mockBinary("gcloud", response{
			startsWith{"auth print-identity-token"},
			simply("F4KE-T0K3N-3B49"),
		}),
		mockBinary("java"),
		mockBinary("mvn"),
		mockBinary("ko"),
		mockBinary("cosign"),
		mockBinary("rcodesign"),
		mockBinary("gsutil"),
		mockBinary("kubetest2"),
	)
}

func union(scriptlets ...scriptlet) scriptlet {
	return fnScriptlet(func(t TestingT) string {
		code := make([]string, 0, len(scriptlets)*10)
		for _, s := range scriptlets {
			code = append(code, s.scriptlet(t))
		}
		return strings.Join(code, "\n")
	})
}

type TestingT interface {
	TempDir() string
	Logf(format string, args ...interface{})
	require.TestingT
}

func (s shellScript) run(t TestingT, commands []string) (int, string, string, string) {
	s.prefetch(t)
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
		source += "\n" + strings.TrimPrefix(sclet.scriptlet(t), bashShebang) + "\n"
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
	p := path.Join(dir, fmt.Sprintf("unittest-%s.bash", randString(12)))
	err := os.WriteFile(p, []byte(src), 0o600)
	require.NoError(t, err)
	return p
}

type prefetcher interface {
	prefetch(t TestingT)
}

type fnPrefetcher func(t TestingT)

func (f fnPrefetcher) prefetch(t TestingT) {
	f(t)
}

// goRunHelpPrefetcher will call `go run tool --help` before the testing starts.
// This is to ensure the given tool is downloaded and compiled, so the download
// and compilation messages, which go prints will not influence the test.
func goRunHelpPrefetcher(tool string) prefetcher {
	return fnPrefetcher(func(t TestingT) {
		c := exec.Command("go", "run", tool, "--help")
		var (
			stdout, stderr io.ReadCloser
			err            error
		)
		stdout, err = c.StdoutPipe()
		require.NoError(t, err)
		stderr, err = c.StderrPipe()
		require.NoError(t, err)
		err = c.Run()
		if err != nil {
			stdBytes, merr := io.ReadAll(stdout)
			require.NoError(t, merr)
			errBytes, rerr := io.ReadAll(stderr)
			require.NoError(t, rerr)
			require.NoError(t, err,
				"------\nSTDOUT\n------", string(stdBytes),
				"------\nSTDERR\n------", string(errBytes))
		}
	})
}

type prefetchScriptlet struct {
	delegate    scriptlet
	prefetchers []prefetcher
}

func (p prefetchScriptlet) scriptlet(t TestingT) string {
	return p.delegate.scriptlet(t)
}

func (p prefetchScriptlet) prefetch(t TestingT) {
	for _, pr := range p.prefetchers {
		pr.prefetch(t)
	}
}

func (s shellScript) prefetch(t TestingT) {
	for _, sclet := range s.scriptlets {
		if pf, ok := sclet.(prefetcher); ok {
			pf.prefetch(t)
		}
	}
}

func currentDir() string {
	_, file, _, _ := runtime.Caller(0)
	return path.Dir(file)
}

func randString(n int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
