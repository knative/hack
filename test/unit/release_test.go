package unit_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestReleaseHelperFunctions(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `major_minor_version "v0.2.1"`,
		stdout: lines("0.2"),
	}, {
		name:   `major_minor_version "0.2.1"`,
		stdout: lines("0.2"),
	}, {
		name:   `patch_version "v0.2.1"`,
		stdout: lines("1"),
	}, {
		name:   `patch_version "0.2.1"`,
		stdout: lines("1"),
	}, {
		name:   `hash_from_tag "v20010101-deadbeef"`,
		stdout: lines("deadbeef"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingVersion(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --version`,
		stderr: aborted("missing parameter after --version"),
	}, {
		name:   `parse_flags --version a`,
		stderr: aborted("version format must be '[0-9].[0-9].[0-9]'"),
	}, {
		name:   `parse_flags --version 0.0`,
		stderr: aborted("version format must be '[0-9].[0-9].[0-9]'"),
	}, {
		name:   `parse_flags --version 1.0.0`,
		stdout: empty(),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingBranch(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --branch`,
		stderr: aborted("missing parameter after --branch"),
	}, {
		name:   `parse_flags --branch a`,
		stderr: aborted("branch name must be 'release-[0-9].[0-9]'"),
	}, {
		name:   `parse_flags --branch 0.0`,
		stderr: aborted("branch name must be 'release-[0-9].[0-9]'"),
	}, {
		name:   `parse_flags --branch release-0.0`,
		stdout: empty(),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingReleaseNotes(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tmpfile := t.TempDir() + "/release-notes.md"
	err := os.WriteFile(tmpfile,
		[]byte("# Release Notes\n\n## 1.0.0\n\n* First release\n"),
		0o600)
	require.NoError(t, err)
	tcs := []testCase{{
		name:   `parse_flags --release-notes`,
		stderr: aborted("missing parameter after --release-notes"),
	}, {
		name:   `parse_flags --release-notes a`,
		stderr: aborted("file a doesn't exist"),
	}, {
		name:     `parse_flags --release-notes release-notes.md`,
		commands: []string{fmt.Sprintf(`parse_flags --release-notes %s`, tmpfile)},
		stdout:   empty(),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingReleaseGcsGcr(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --release-gcs`,
		stderr: aborted("missing parameter after --release-gcs"),
	}, {
		name:   `parse_flags --release-gcs a --publish`,
		stdout: empty(),
	}, {
		name:   `parse_flags --release-gcr`,
		stderr: aborted("missing parameter after --release-gcr"),
	}, {
		name:   `parse_flags --release-gcr a --publish`,
		stdout: empty(),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingReleaseConstraints(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --dot-release --auto-release`,
		stderr: aborted("cannot have both --dot-release and --auto-release set simultaneously"),
	}, {
		name:   `parse_flags --auto-release --version 1.0.0`,
		stderr: aborted("cannot have both --version and --auto-release set simultaneously"),
	}, {
		name:   `parse_flags --auto-release --branch release-0.0`,
		stderr: aborted("cannot have both --branch and --auto-release set simultaneously"),
	}, {
		name:   `parse_flags --release-gcs a --release-dir b`,
		stderr: aborted("cannot have both --release-gcs and --release-dir set simultaneously"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingNightly(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --from-nightly`,
		stderr: aborted("missing parameter after --from-nightly"),
	}, {
		name:   `parse_flags --from-nightly aaa`,
		stderr: aborted("nightly tag must be 'vYYYYMMDD-commithash'"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingGithubToken(t *testing.T) {
	t.Parallel()
	tmpfile := t.TempDir() + "/github.token"
	token := rand.String(12)
	err := os.WriteFile(tmpfile, []byte(token+"\n"), 0o600)
	require.NoError(t, err)
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --github-token`,
		stderr: aborted("missing parameter after --github-token"),
	}, {
		name:   `parse_flags --github-token github.token`,
		stdout: lines(token),
		commands: []string{
			fmt.Sprintf(`parse_flags --github-token %s`, tmpfile),
			`echo $GITHUB_TOKEN`,
		},
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingGcsGcrIgnoredValues(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name:   `parse_flags --release-gcs foo`,
		stdout: lines("Not publishing the release, GCS flag is ignored"),
	}, {
		name:   `parse_flags --release-gcr foo`,
		stdout: lines("Not publishing the release, GCR flag is ignored"),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseFlagParsingDefaults(t *testing.T) {
	t.Parallel()
	sc := newShellScript(loadFile("source-release.bash"))
	tcs := []testCase{{
		name: `parse_flags`,
		commands: []string{
			"parse_flags",
			`echo :${KO_DOCKER_REPO}:`,
			`echo :${RELEASE_GCS_BUCKET}:`,
		},
		stdout: lines(
			":ko.local:",
			"::",
		),
	}, {
		name: `parse_flags --publish --release-dir foo`,
		commands: []string{
			"parse_flags --publish",
			`echo :${KO_DOCKER_REPO}:`,
			`echo :${RELEASE_GCS_BUCKET}:${RELEASE_DIR}:`,
		},
		stdout: lines(
			":gcr.io/knative-nightly:",
			":knative-nightly/hack::",
		),
	}, {
		name: `parse_flags --release-gcr foo --publish`,
		commands: []string{
			"parse_flags --release-gcr foo --publish",
			`echo :${KO_DOCKER_REPO}:`,
			`echo :${RELEASE_GCS_BUCKET}:${RELEASE_DIR}:`,
		},
		stdout: lines(
			":foo:",
			":knative-nightly/hack::",
		),
	}, {
		name: `parse_flags --release-gcs foo --publish`,
		commands: []string{
			"parse_flags --release-gcs foo --publish",
			`echo :${KO_DOCKER_REPO}:`,
			`echo :${RELEASE_GCS_BUCKET}:${RELEASE_DIR}:`,
		},
		stdout: lines(
			":gcr.io/knative-nightly:",
			":foo::",
		),
	}}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}
