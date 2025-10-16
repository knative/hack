package unit_test

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/abiosoft/lineprefix"
	"github.com/charmbracelet/gum/style"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thanhpk/randstr"
)

var (
	//go:embed scripts/*.bash
	scripts      embed.FS
	bashQuotesRx = regexp.MustCompile("(?m)^#.*")
	toMuchNlRx   = regexp.MustCompile("(?m)\n{3,}")
)

func TestMain(m *testing.M) {
	ensureGoModDownloaded()
	os.Exit(m.Run())
}

// ensureGoModDownloaded will download the go modules before running the tests
// to avoid the download and compilation messages, which may influence the
// output assertions.
func ensureGoModDownloaded() {
	fmt.Println("Pre-fetching go modules, to avoid download messages during tests...")
	cmd := exec.Command("go", "mod", "download", "-x")
	cmd.Dir = path.Dir(currentDir())
	cmd.Stdout = lineprefix.New(
		lineprefix.Writer(os.Stdout),
		lineprefix.Color(color.New(color.FgCyan)),
		lineprefix.Prefix("STDOUT |"),
	)
	cmd.Stderr = lineprefix.New(
		lineprefix.Writer(os.Stderr),
		lineprefix.Color(color.New(color.FgRed)),
		lineprefix.Prefix("STDERR |"),
	)
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func aborted(msg string) []check {
	fmsg := fmt.Sprintf("ERROR: %s", msg)
	styles := libglossDefaults(style.StylesNotHidden{
		Border:           "double",
		Align:            "center",
		Padding:          "1 3",
		Foreground:       "#D00",
		BorderForeground: "#D00",
	})
	return []check{contains(makeBanner(styles, fmsg))}
}

func warned(msg string) []check {
	fmsg := fmt.Sprintf("WARN: %s", msg)
	styles := libglossDefaults(style.StylesNotHidden{
		Border:           "rounded",
		Align:            "center",
		Padding:          "1 3",
		Foreground:       "#DD0",
		BorderForeground: "#DD0",
	})

	return []check{contains(makeBanner(styles, fmsg))}
}

func header(msg string) check {
	styles := libglossDefaults(style.StylesNotHidden{
		Border:           "double",
		Align:            "center",
		Padding:          "1 3",
		Foreground:       "45",
		BorderForeground: "45",
	})
	return contains(makeBanner(styles, msg))
}

func subheader(msg string) check {
	styles := libglossDefaults(style.StylesNotHidden{
		Border:           "rounded",
		Align:            "center",
		Padding:          "0 1",
		Foreground:       "44",
		BorderForeground: "44",
	})
	return contains(makeBanner(styles, msg))
}

func libglossDefaults(styles style.StylesNotHidden) style.StylesNotHidden {
	if styles.Align == "" {
		styles.Align = "left"
	}
	if styles.Border == "" {
		styles.Border = "none"
	}
	if styles.Margin == "" {
		styles.Margin = "0 0"
	}
	if styles.Padding == "" {
		styles.Padding = "0 0"
	}
	return styles
}

func makeBanner(styles style.StylesNotHidden, msg string) string {
	return styles.ToLipgloss().Render(msg+"\n\nat 2018-07-18 23:00:00") + "\n"
}

func empty() []check {
	return equal("")
}

func lines(strs ...string) []check {
	return equal(strings.Join(strs, "\n") + "\n")
}

func contains(str string) check {
	return func(t TestingT, output string, otype outputType) bool {
		assert.Contains(t, output, str,
			"The %s does not contain:\n%v",
			otype, str)
		return strings.Contains(output, str)
	}
}

func equal(str string) []check {
	return []check{func(t TestingT, output string, otype outputType) bool {
		assert.Equal(t, str, output,
			"The %s wasn't equal to:\n%v",
			otype, str)
		return str == output
	}}
}

func dumpOutput(output string, otype outputType) string {
	label := strings.ToUpper(string(otype))
	return fmt.Sprintf("\n───── BEGIN %s ─────\n%v────── END %s ──────\n",
		label, output, label)
}

type outputType string

const (
	outputTypeStdout outputType = "stdout"
	outputTypeStderr outputType = "stderr"
)

type check func(t TestingT, output string, otype outputType) bool

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

var errOutStripGoSwitchingRe = regexp.MustCompile(
	"go: knative\\.dev/toolbox@v0\\.0\\.0-\\d+-[0-9a-f]+ requires " +
		"go >= \\d\\.\\d+\\.\\d+; switching to go\\d\\.\\d+\\.\\d+\n",
)

func (tc testCase) test(sc shellScript) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		code, out, errOut, src := sc.run(t, tc.testCommands())
		tc.validRetcode(t, code)
		checkStream := func(output string, otype outputType, checks []check) {
			success := true
			for _, chck := range checks {
				success = success && chck(t, output, otype)
			}
			if !success {
				t.Logf("Printing %s because of failed check:%s", otype,
					dumpOutput(output, otype))
			}
		}
		checkStream(out, outputTypeStdout, coalesce(tc.stdout, empty()))
		// skip go switching messages from asserting
		errOut = errOutStripGoSwitchingRe.ReplaceAllString(errOut, "")
		checkStream(errOut, outputTypeStderr, coalesce(tc.stderr, empty()))

		if t.Failed() {
			failedScriptPath := path.Join(os.TempDir(),
				t.Name(),
				time.Now().Format("20060102-150405")+".bash")
			require.NoError(t, os.MkdirAll(path.Dir(failedScriptPath), 0o755))
			require.NoError(t, os.WriteFile(failedScriptPath, []byte(src), 0o755))
			t.Logf("The script that failed: %s", failedScriptPath)
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
	label := "Retcode mismatch"
	if tc.retcode != nil {
		assert.Equal(t, int(*tc.retcode), gotRetcode, label)
		return
	}
	if len(tc.stderr) > 0 {
		assert.NotEqual(t, 0, gotRetcode, label)
	} else {
		assert.Equal(t, 0, gotRetcode, label)
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
		append([]scriptlet{mockBinary("date", response{
			anyArgs{}, simply("2018-07-18 23:00:00"),
		})}, scriptlets...),
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
	binPath, err := findExecutable(bin)
	if err != nil {
		panic(err)
	}
	return []string{
		fmt.Sprintf(`  '%s' "$@"`, binPath),
	}
}

func findExecutable(bin string) (string, error) {
	goroot := runtime.GOROOT()
	binPath := filepath.Join(goroot, "bin", bin)
	if runtime.GOOS == "windows" {
		binPath = binPath + ".exe"
	}
	if err := checkExecutable(binPath); err != nil {
		binPath, err = exec.LookPath(bin)
		if err != nil {
			return "", err
		}
	}
	return binPath, nil
}

func checkExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	m := d.Mode()
	if m.IsDir() {
		return syscall.EISDIR
	}
	if m&0111 != 0 {
		return nil
	}
	return fs.ErrPermission
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
			// The ghost icon is used to differentiate the mocked command output
			// from the real one.
			fmt.Sprintf(`echo "👻 %s $*"`, name),
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
	gum := "github.com/charmbracelet/gum@v0.14.1"
	callOriginals := []args{
		startsWith{"run " + lstags},
		startsWith{"run " + modscope},
		startsWith{"run " + gum},
		startsWith{"run ./"},
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
			goRunHelpPrefetcher(gum),
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
	Cleanup(func())
	Parallel()
	Failed() bool
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
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			require.NoError(t, err)
		}
	}
	return c.ProcessState.ExitCode(), bo.String(), be.String(), src
}

func (s shellScript) source(t TestingT, commands []string) string {
	goroot := runtime.GOROOT()
	source := fmt.Sprintf(`
set -Eeuo pipefail
export TMPPATH='%s'
export PATH="${TMPPATH}:%s:${PATH}"
export KNATIVE_HACK_SCRIPT_MANUAL_VERBOSE=true
`, t.TempDir(), fmt.Sprintf("%s/bin", goroot))
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
	p := path.Join(dir, fmt.Sprintf("unittest-%s.bash", randstr.String(12)))
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
		stdout := bytes.NewBuffer(make([]byte, 0, 1024))
		stderr := bytes.NewBuffer(make([]byte, 0, 1024))
		gobin, err := findExecutable("go")
		require.NoError(t, err)
		c := exec.Command(gobin, "run", tool, "--help")
		c.Env = append(os.Environ(), "GOTOOLCHAIN=auto")
		c.Stdout = stdout
		c.Stderr = stderr
		err = c.Run()
		if err != nil {
			errBytes := stderr.Bytes()
			stdBytes := stdout.Bytes()
			require.NoError(t, err,
				"───── BEGIN STDOUT ─────\n%s\n────── END STDOUT ──────\n"+
					"───── BEGIN STDERR ─────\n%s\n────── END STDERR ──────",
				string(stdBytes), string(errBytes))
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
