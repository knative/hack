package unit_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thanhpk/randstr"
)

var CantFindChecksums = warned("cannot find checksums file")

func TestBuildFromSource(t *testing.T) {
	t.Parallel()

	outChecks := []check{
		contains("Signing Images with the identity signer@knative-releases.test"),
		contains("Signing checksums with the identity signer@knative-releases.test"),
		contains("Notarizing macOS Binaries for the release"),
	}
	checksumsContent := `🧮     Post Notarization Checksum:
4d410c6611b89b21215e06046dc8104aa668c8e93a5b73062e45bd43c6c422cc  foo-linux-amd64
6fedd2d0b79cbd3faf11f159f6b229707e191a5bcc5f727fd33b916d517c8ed4  foo-linux-arm64
58eaa00b44cb836d09f009791bdb2c521afc18f7a2dac80422a6204774d6a677  foo-linux-ppc64le
7b33c5e58372290a7addc5e9b95a1fef33bb1ce38660dd4fdc65b9862e466a59  foo-linux-s390x
9ee0670b6715542ef64a336ae68342fde32d3045273dcfe67d97c22f72f4c039  foo-darwin-amd64
74da512cfed7a90713a7161f34a2339fe2e9c9cec8bd3cb30566c464bf2c18f1  foo-darwin-arm64
73517e997b68696b1a6be4957519b800e26c9bc44c1b7f46fe90be0834d1af07  foo-windows-amd64.exe
9ac630646ca5b77fbf716f9a780d33f26357bbd8b242c14e0863cdde72aacbf0  foo.yaml
`
	tcs := []testCase{{
		name:    "build_from_source",
		retcode: retcode(0),
		stderr:  CantFindChecksums,
		stdout:  outChecks,
	}, {
		name: "build_from_source (with_checksums)",
		commands: []string{
			`export CALCULATE_CHECKSUMS=1`,
			`build_from_source`,
		},
		stdout: append(outChecks, contains(checksumsContent)),
	}}
	for i := range tcs {
		tc := tcs[i]
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			sc := testReleaseShellScript(
				envs(map[string]string{
					"BUILD_DIR":                    tmp,
					"APPLE_CODESIGN_KEY":           randomFile(t, tmp, "codesign.key"),
					"APPLE_CODESIGN_PASSWORD_FILE": randomFile(t, tmp, "codesign.pass"),
					"APPLE_NOTARY_API_KEY":         randomFile(t, tmp, "notary.key"),
					"SIGNING_IDENTITY":             "signer@knative-releases.test",
				}),
				loadFile("fake-build-release.bash"),
			)
			tc.test(sc)(t)
		})
	}
}

func TestFindChecksumsFile(t *testing.T) {
	t.Parallel()

	foundChecksums := lines("/tmp/other/checksums.txt")

	tcs := []testCase{{
		name:    "find_checksums_file /tmp/file1.out /tmp/file2.out",
		retcode: retcode(0),
		stderr:  CantFindChecksums,
		stdout:  empty(),
	}, {
		name:   "find_checksums_file /tmp/file1.out /tmp/other/checksums.txt /tmp/file2.out",
		stdout: foundChecksums,
	}, {
		name: `find_checksums_file "$ARTIFACTS_TO_PUBLISH"`,
		commands: []string{
			`export ARTIFACTS_TO_PUBLISH="/tmp/file1.out /tmp/other/checksums.txt /tmp/file2.out"`,
			`find_checksums_file "$ARTIFACTS_TO_PUBLISH"`,
		},
		stdout: foundChecksums,
	}, {
		name:    `find_checksums_file "$ARTIFACTS_TO_PUBLISH" # without checksums in artifacts`,
		retcode: retcode(0),
		commands: []string{
			`export ARTIFACTS_TO_PUBLISH="/tmp/file1.out /tmp/file2.out"`,
			`find_checksums_file "$ARTIFACTS_TO_PUBLISH"`,
		},
		stderr: CantFindChecksums,
		stdout: empty(),
	}, {
		name: `find_checksums_file "$ARTIFACTS_TO_PUBLISH" # with double spaces`,
		commands: []string{
			`export ARTIFACTS_TO_PUBLISH="/tmp/file1.out  /tmp/other/checksums.txt /tmp/file2.out"`,
			`find_checksums_file "$ARTIFACTS_TO_PUBLISH"`,
		},
		stdout: foundChecksums,
	}}
	sc := testReleaseShellScript(loadFile("fake-build-release.bash"))
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, tc.test(sc))
	}
}

func TestReleaseHelperFunctions(t *testing.T) {
	t.Parallel()
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	token := randstr.String(12)
	err := os.WriteFile(tmpfile, []byte(token+"\n"), 0o600)
	require.NoError(t, err)
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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
	sc := testReleaseShellScript()
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

func testReleaseShellScript(scrps ...scriptlet) shellScript {
	aargs := make([]scriptlet, 0, len(scrps)+3)
	aargs = append(aargs,
		fakeProwJob(),
		loadFile("source-release.bash"),
		loadFile("fake-presubmit-tests.bash"),
	)
	aargs = append(aargs, scrps...)
	return newShellScript(aargs...)
}

func randomFile(tb testing.TB, tmpdir string, filename string) string {
	r := randstr.String(24)
	fp := path.Join(tmpdir, filename)
	if err := ioutil.WriteFile(fp, []byte(r), 0o600); err != nil {
		tb.Fatal(err)
	}
	return fp
}
