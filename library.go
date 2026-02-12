package hack

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const KNATIVE_TESTS_PROJECT = "knative-tests"

var (
	IS_PROW       			      bool
	GOPATH        			      string
	REPO_ROOT_DIR 			      string
	REPO_NAME    			      string
	IS_LINUX      			      bool
	IS_OSX        				  bool
	IS_WINDOWS   				  bool
	_TEST_INFRA_SCRIPTS_DIR       string
	REPO_NAME_FORMATTED 		  string
	KNATIVE_SERVING_RELEASE_CRDS  string
	KNATIVE_SERVING_RELEASE_CORE  string
	KNATIVE_NET_ISTIO_RELEASE     string
	KNATIVE_EVENTING_RELEASE      string
)

func init() {
	if _, ok := os.LookupEnv("PROW_JOB_ID"); ok {
		IS_PROW = true
	} else {
		IS_PROW = false
	}

	if _, ok := os.LookupEnv("REPO_ROOT_DIR"); !ok {
		repoRootDir, err := os.Executable()
		if err == nil {
			REPO_ROOT_DIR = repoRootDir
		}
	}

	if _, ok := os.LookupEnv("GOPATH"); !ok {
		GOPATH = os.Getenv("GOPATH")
		if GOPATH == "" {
			GOPATH = os.Getenv("HOME") + "/go"
		}
	}

	if _, ok := os.LookupEnv("REPO_ROOT_DIR"); !ok {
		repoRootDir, err := os.Executable()
		if err == nil {
			REPO_ROOT_DIR = repoRootDir
		}
	}

	REPO_NAME = resolveRepoName(REPO_ROOT_DIR)

	switch runtime.GOOS {
	case "linux":
		IS_LINUX = true
	case "darwin":
		IS_OSX = true
	case "windows":
		IS_WINDOWS = true
	default:
		fmt.Println("** Internal error in library.go, unknown OS", runtime.GOOS)
		os.Exit(1)
	}

	_TEST_INFRA_SCRIPTS_DIR, _  	 = getCanonicalPath(getScriptPath())
	REPO_NAME_FORMATTED				 = fmt.Sprintf("Knative %s", capitalize(replaceDashWithSpace(REPO_NAME)))
	KNATIVE_SERVING_RELEASE_CRDS, _  = GetLatestKnativeYAMLSource("serving", "serving-crds")
	KNATIVE_SERVING_RELEASE_CORE, _  = GetLatestKnativeYAMLSource("serving", "serving-core")
	KNATIVE_NET_ISTIO_RELEASE, _     = GetLatestKnativeYAMLSource("net-istio", "net-istio")
	KNATIVE_EVENTING_RELEASE, _      = GetLatestKnativeYAMLSource("eventing", "eventing")

}

// Replace dashes with spaces in a string.
func replaceDashWithSpace(s string) string {
	return strings.ReplaceAll(s, "-", " ")
}

// Get the absolute path of the current script file.
func getScriptPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	scriptPath := filepath.Dir(executable)
	return scriptPath, nil
}

func resolveRepoName(rootDir string) string {
	repoName := strings.TrimSuffix(filepath.Base(rootDir), "-sandbox")
	repoName = strings.TrimPrefix(repoName, "knative-")
	return repoName
}

func majorVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	tokens := strings.Split(version, ".")
	return tokens[0]
}

func minorVersion(version string) string {
	tokens := strings.Split(version, ".")
	return tokens[1]
}

func hashCode(input string) int {
	var h int
	for i := 0; i < len(input); i++ {
		val := int(input[i])
		hval := 31*h + val
		if hval > 2147483647 {
			h = (hval - 2147483648) % 2147483648
		} else if hval < -2147483648 {
			h = (hval + 2147483648) % 2147483648
		} else {
			h = hval
		}
	}
	return h
}

func calcRetcode(args ...string) int {
	rc := 1
	rcc := hashCode(concatenateStrings(args...))
	if rcc != 0 {
		rc = rcc % 255
	}
	return rc
}

func concatenateStrings(strings ...string) string {
	var result string
	for _, str := range strings {
		result += str
	}
	return result
}

func abort(args ...string) {
	msg := fmt.Sprintf("ERROR: %v", args)
	makeBanner('*', msg)
	abortRetcode := calcRetcode(args...)
	os.Exit(abortRetcode)
}

// makeBanner displays a box banner.
// Parameters:
//
//	char: Character to use for the box.
//	message: Banner message.
func makeBanner(char rune, message string) {
	msg := strings.Repeat(string(char), 4) + " " + message + " " + strings.Repeat(string(char), 4)
	border := strings.Map(func(r rune) rune {
		if r == char {
			return r
		}
		return -1
	}, msg)
	fmt.Printf("%s\n%s\n%s\n", border, msg, border)
	if IS_PROW {
		fmt.Printf("%c%c%c%c %s\n%s\n", char, char, char, char, time.Now().UTC().Format(time.RFC3339Nano), border)
	}
}

// header displays a simple header for logging purposes.
// Parameters:
//
//	text: Text for the header.
func header(text string) {
	upper := strings.ToUpper(text)
	makeBanner('=', upper)
}

// subheader displays a simple subheader for logging purposes.
// Parameters:
//
//	text: Text for the subheader.
func subheader(text string) {
	makeBanner('-', text)
}

// warning displays a simple warning banner for logging purposes.
// Parameters:
//
//	message: Warning message.
func warning(message string) {
	makeBanner('!', "WARN: "+message)
}

// functionExists checks whether the given function name exists.
// Parameters:
//
//	functionName: Name of the function to check.
//
// Returns:
//
//	bool: true if the function exists, false otherwise.
func functionExists(functionName string) bool {
	_, found := reflect.TypeOf(FunctionName{}).MethodByName(functionName)
	return found
}

// Global variable to track group status
var groupTracker string

// GitHub Actions aware output grouping.
func group() {
	// End the group if there is already a group.
	if groupTracker == "" {
		groupTracker = "grouping"
		defer endGroup()
	} else {
		endGroup()
	}
	// Start a new group.
	startGroup(os.Args[1:])
}

// GitHub Actions aware output grouping.
func startGroup(args []string) {
	if workflow := os.Getenv("GITHUB_WORKFLOW"); workflow != "" {
		fmt.Printf("::group::%s\n", strings.Join(args, " "))
		defer endGroup()
	} else {
		fmt.Println("---", strings.Join(args, " "))
	}
}

// GitHub Actions aware end of output grouping.
func endGroup() {
	if workflow := os.Getenv("GITHUB_WORKFLOW"); workflow != "" {
		fmt.Println("::endgroup::")
	}
}

// Waits until the given object doesn't exist.
// Parameters: kind - the kind of the object.
//
//	name - object's name.
//	namespace - namespace (optional).
func waitUntilObjectDoesNotExist(kind, name, namespace string) error {
	kubectlArgs := []string{"get", kind, name}
	description := fmt.Sprintf("%s %s", kind, name)

	if namespace != "" {
		kubectlArgs = append([]string{"get", "-n", namespace, kind, name})
		description = fmt.Sprintf("%s %s/%s", kind, namespace, name)
	}

	fmt.Printf("Waiting until %s does not exist", description)

	for i := 1; i <= 150; i++ { // timeout after 5 minutes
		cmd := exec.Command("kubectl", kubectlArgs...)
		err := cmd.Run()
		if err != nil {
			// Check if the kubectl command exited with a non-zero status code
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 1 {
					fmt.Printf("\n%s does not exist\n", description)
					return nil
				}
			}
			return fmt.Errorf("failed to execute kubectl command: %w", err)
		}

		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n\nERROR: timeout waiting for %s not to exist\n", description)

	// Print the kubectl command output in case of error
	cmd := exec.Command("kubectl", kubectlArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	return fmt.Errorf("timeout waiting for %s not to exist", description)
}

// Waits until the given object exists.
// Parameters: kind - the kind of the object.
//
//	name - object's name.
//	namespace - namespace (optional).
func waitUntilObjectExists(kind, name, namespace string) error {
	kubectlArgs := []string{"get", kind, name}
	description := fmt.Sprintf("%s %s", kind, name)

	if namespace != "" {
		kubectlArgs = append([]string{"get", "-n", namespace, kind, name})
		description = fmt.Sprintf("%s %s/%s", kind, namespace, name)
	}

	fmt.Printf("Waiting until %s exists", description)

	for i := 1; i <= 150; i++ { // timeout after 5 minutes
		cmd := exec.Command("kubectl", kubectlArgs...)
		err := cmd.Run()
		if err == nil {
			fmt.Printf("\n%s exists\n", description)
			return nil
		}

		// Check if the kubectl command exited with a non-zero status code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				fmt.Print(".")
				time.Sleep(2 * time.Second)
				continue
			}
		}

		return fmt.Errorf("failed to execute kubectl command: %w", err)
	}

	fmt.Printf("\n\nERROR: timeout waiting for %s to exist\n", description)

	// Print the kubectl command output in case of error
	cmd := exec.Command("kubectl", kubectlArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	return fmt.Errorf("timeout waiting for %s to exist", description)
}

// Waits until all pods are running in the given namespace.
// Parameters: namespace - the namespace.
func waitUntilPodsRunning(namespace string) error {
	fmt.Printf("Waiting until all pods in namespace %s are up", namespace)

	var failedPod string

	for i := 1; i <= 150; i++ { // timeout after 5 minutes
		// List all pods. Ignore Terminating pods.
		cmd := exec.Command("kubectl", "get", "pods", "--no-headers", "-n", namespace)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to execute kubectl command: %w", err)
		}

		pods := string(output)
		notRunningPods := getNotRunningPods(pods)

		if len(pods) > 0 && len(notRunningPods) == 0 {
			// All Pods are running or completed. Verify the containers on each Pod.
			allReady := true

			lines := strings.Split(pods, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}

				fields := strings.Fields(line)
				podName := fields[0]
				status := strings.Split(fields[1], "/")

				// Set this Pod as the failedPod. If nothing is wrong with it, then after the checks, set
				// failedPod to an empty string.
				failedPod = podName

				if len(status) < 2 || status[0] == "" || status[1] == "" ||
					status[0] == "0" || status[1] == "0" || status[0] != status[1] {
					allReady = false
					break
				}

				// All the checks passed, this is not a failed pod.
				failedPod = ""
			}

			if allReady {
				fmt.Printf("\nAll pods are up:\n%s\n", pods)
				return nil
			}
		} else if len(notRunningPods) > 0 {
			// At least one Pod is not running, just save the first one's name as the failedPod.
			failedPod = strings.Fields(notRunningPods)[0]
		}

		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n\nERROR: timeout waiting for pods to come up\n%s\n", pods)

	if failedPod != "" {
		fmt.Printf("\n\nFailed Pod (data in YAML format) - %s\n\n", failedPod)
		cmd := exec.Command("kubectl", "-n", namespace, "get", "pods", failedPod, "-oyaml")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()

		fmt.Println("\n\nPod Logs\n")
		cmd = exec.Command("kubectl", "-n", namespace, "logs", failedPod, "--all-containers")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	return fmt.Errorf("timeout waiting for pods to come up")
}

// Helper function to extract not running pods from the pod list.
func getNotRunningPods(pods string) string {
	lines := strings.Split(pods, "\n")
	var notRunningPods []string

	for _, line := range lines {
		if strings.Contains(line, "Running") ||
			strings.Contains(line, "Completed") ||
			strings.Contains(line, "ErrImagePull") ||
			strings.Contains(line, "ImagePullBackOff") {
			continue
		}

		notRunningPods = append(notRunningPods, line)
	}

	return strings.Join(notRunningPods, "\n")
}

// Waits until all batch jobs complete in the given namespace.
// Parameters: namespace - the namespace.
func waitUntilBatchJobComplete(namespace string) error {
	fmt.Printf("Waiting until all batch jobs in namespace %s run to completion.\n", namespace)

	cmd := exec.Command("kubectl", "wait", "job", "--for=condition=Complete", "--all", "-n", namespace, "--timeout=5m")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute kubectl command: %w", err)
	}

	if strings.TrimSpace(string(output)) != "" {
		return fmt.Errorf("timeout waiting for batch jobs to complete")
	}

	return nil
}

// Waits until the given service has an external address (IP/hostname).
// Parameters: namespace - the namespace.
//
//	serviceName - the service name.
func waitUntilServiceHasExternalIP(namespace, serviceName string) error {
	fmt.Printf("Waiting until service %s in namespace %s has an external address (IP/hostname)\n", serviceName, namespace)

	for i := 0; i < 150; i++ { // timeout after 15 minutes
		ip, err := getServiceExternalIP(namespace, serviceName)
		if err == nil && ip != "" {
			fmt.Printf("\nService %s.%s has IP %s\n", serviceName, namespace, ip)
			return nil
		}

		hostname, err := getServiceExternalHostname(namespace, serviceName)
		if err == nil && hostname != "" {
			fmt.Printf("\nService %s.%s has hostname %s\n", serviceName, namespace, hostname)
			return nil
		}

		fmt.Print(".")
		time.Sleep(6 * time.Second)
	}

	fmt.Printf("\n\nERROR: timeout waiting for service %s.%s to have an external address\n", serviceName, namespace)
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute kubectl command: %w", err)
	}
	fmt.Println(string(output))
	return fmt.Errorf("timeout waiting for service %s.%s to have an external address", serviceName, namespace)
}

// Retrieves the external IP of the service in the given namespace.
func getServiceExternalIP(namespace, serviceName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "svc", "-n", namespace, serviceName, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute kubectl command: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Retrieves the external hostname of the service in the given namespace.
func getServiceExternalHostname(namespace, serviceName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "svc", "-n", namespace, serviceName, "-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute kubectl command: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// Waits until the given service has an external address (IP/hostname) that allows HTTP connections.
// Parameters: namespace - the namespace.
//
//	serviceName - the service name.
func waitUntilServiceHasExternalHTTPAddress(namespace, serviceName string) error {
	ns := namespace
	svc := serviceName
	sleepSeconds := 6
	attempts := 150

	fmt.Printf("Waiting until service %s/%s has an external address (IP/hostname)\n", ns, svc)
	for attempt := 1; attempt <= attempts; attempt++ {
		address, err := getServiceExternalAddress(ns, svc)
		if err != nil {
			return fmt.Errorf("failed to get service external address: %w", err)
		}

		if address != "" {
			fmt.Printf("Service %s/%s has %s\n", ns, svc, address)

			status, err := probeHTTPStatus(address)
			if err != nil {
				fmt.Printf("%s is not ready: prober encountered an error: %v\n", address, err)
			} else {
				fmt.Printf("%s is ready: prober observed HTTP %d\n", address, status)
				return nil
			}
		}

		fmt.Print(".")
		time.Sleep(time.Duration(sleepSeconds) * time.Second)
	}

	fmt.Printf("\n\nERROR: timeout waiting for service %s/%s to have an external HTTP address\n", ns, svc)
	cmd := exec.Command("kubectl", "get", "pods", "-n", ns)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute kubectl command: %w", err)
	}
	fmt.Println(string(output))
	return fmt.Errorf("timeout waiting for service %s/%s to have an external HTTP address", ns, svc)
}

// Retrieves the external address of the service in the given namespace.
func getServiceExternalAddress(namespace, serviceName string) (string, error) {
	ipCmd := exec.Command("kubectl", "get", "svc", serviceName, "-n", namespace, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
	ipOutput, err := ipCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute kubectl command: %w", err)
	}

	ip := strings.TrimSpace(string(ipOutput))
	if ip != "" {
		return ip, nil
	}

	hostnameCmd := exec.Command("kubectl", "get", "svc", serviceName, "-n", namespace, "-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}")
	hostnameOutput, err := hostnameCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute kubectl command: %w", err)
	}

	hostname := strings.TrimSpace(string(hostnameOutput))
	return hostname, nil
}

// Probes the HTTP status of the given address.
func probeHTTPStatus(address string) (int, error) {
	resp, err := http.Get("http://" + address)
	if err != nil {
		return 0, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// Waits for the endpoint to be routable.
// Parameters: externalIP - the external ingress IP address.
//
//	hostname - the cluster hostname.
func waitUntilRoutable(externalIP, hostname string) error {
	fmt.Printf("Waiting until cluster %s at %s has a routable endpoint\n", hostname, externalIP)
	for i := 1; i <= 150; i++ { // timeout after 5 minutes
		resp, err := http.Get(fmt.Sprintf("http://%s", externalIP))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("Endpoint is now routable")
			return nil
		}
		fmt.Print(".")
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("\n\nERROR: Timed out waiting for endpoint to be routable\n")
	return fmt.Errorf("timed out waiting for endpoint to be routable")
}

// Returns the name of the first pod of the given app.
// Parameters: appName - the app name.
//
//	namespace - the namespace (optional).
func getAppPod(appName, namespace string) (string, error) {
	pods, err := getAppPods(appName, namespace)
	if err != nil {
		return "", err
	}
	if len(pods) > 0 {
		return pods[0], nil
	}
	return "", fmt.Errorf("no pods found for app: %s", appName)
}

// Returns the name of all pods of the given app.
// Parameters: appName - the app name.
//
//	namespace - the namespace (optional).
func getAppPods(appName, namespace string) ([]string, error) {
	kubectlArgs := []string{"get", "pods"}
	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}
	kubectlArgs = append(kubectlArgs, "--selector=app="+appName, "--output=jsonpath={.items[*].metadata.name}")
	cmd := exec.Command("kubectl", kubectlArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	podNames := strings.TrimSpace(string(output))
	if podNames == "" {
		return nil, fmt.Errorf("no pods found for app: %s", appName)
	}
	return strings.Split(podNames, " "), nil
}

// Capitalizes the first letter of each word.
// Parameters: words - the words to capitalize.
func capitalize(words ...string) string {
	capitalized := make([]string, len(words))
	for i, word := range words {
		if len(word) > 0 {
			initial := strings.ToUpper(string(word[0]))
			capitalized[i] = initial + word[1:]
		} else {
			capitalized[i] = word
		}
	}
	return strings.Join(capitalized, " ")
}

// Dumps pod logs for the given app.
// Parameters: app - the app name.
//
//	namespace - the namespace.
func dumpAppLogs(app, namespace string) {
	fmt.Printf(">>> %s %s logs:\n", repoNameFormatted, app)
	pods, _ := getAppPods(app, namespace)
	for _, pod := range pods {
		fmt.Printf(">>> Pod: %s\n", pod)
		cmd := exec.Command("kubectl", "-n", namespace, "logs", pod, "--all-containers")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

// Print failed step.
// Parameters: steps - description of step that failed.
func stepFailed(steps ...string) {
	fmt.Printf("Step failed: %s\n", steps)
}

// Create a temporary file with the given extension in a way that works on both Linux and macOS.
// Parameters: namePrefix - file name without extension (e.g. 'myfile_XXXX')
//
//	extension - file extension (e.g. 'xml')
//
// Returns: string - the path of the created temporary file.
func mktempWithExtension(namePrefix string, extension string) (string, error) {
	tmpFile, err := ioutil.TempFile("", namePrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	fullName := tmpFile.Name() + "." + extension
	err = os.Rename(tmpFile.Name(), fullName)
	if err != nil {
		return "", fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return fullName, nil
}

// Create a JUnit XML for a test.
// Parameters: checkClassName - check class name as an identifier (e.g. BuildTests)
//
//	checkName - check name as an identifier (e.g., GoBuild)
//	failureMsg - failure message (can contain newlines), optional (means success)
//
// Returns: error - any error encountered during the operation.
func createJUnitXML(checkClassName string, checkName string, failureMsg string) error {
	xmlFile, err := mktempWithExtension(filepath.Join(artifactsDir, "junit_XXXXXXXX"), "xml")
	if err != nil {
		return err
	}
	fmt.Printf("XML report for %s::%s written to %s\n", checkClassName, checkName, xmlFile)

	err = runKntest("junit", "--suite="+checkClassName, "--name="+checkName, "--err-msg="+failureMsg, "--dest="+xmlFile)
	if err != nil {
		return fmt.Errorf("failed to run kntest: %w", err)
	}

	return nil
}

// Run the specified Go package.
// Parameters: packageVersion - Go package with version (e.g., knative.dev/test-infra/tools/kntest/cmd/kntest@latest)
//
//	params - parameters passed to the package.
func goRun(packageVersion string, params ...string) error {
	cmd := exec.Command("go", "run", packageVersion)
	cmd.Args = append(cmd.Args, params...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run Go package: %w", err)
	}
	return nil
}

// Run the kntest command with the specified parameters.
// Parameters: params - parameters passed to the kntest command.
func runKntest(params ...string) error {
	return goRun("kntest.dev/test-infra/tools/kntest/cmd/kntest@latest", params...)
}

// Runs a Go test and generates a JUnit summary.
// Parameters: params - parameters passed to 'go test'
func reportGoTest(params ...string) error {
	goTestArgs := params
	xmlFile, err := mktempWithExtension(filepath.Join(artifactsDir, "junit_XXXXXXXX"), "xml")
	if err != nil {
		return err
	}
	logFile := strings.ReplaceAll(xmlFile, "junit_", "go_test_")
	logFile = strings.ReplaceAll(logFile, ".xml", ".jsonl")
	fmt.Printf("Running go test with args: %s\n", strings.Join(goTestArgs, " "))

	GO_

	gotestRetCode := 0
	err = goRun("gotest.tools/gotestsum@v1.8.0",
		"--format", GO_TEST_VERBOSITY,
		"--junitfile", xmlFile,
		"--junitfile-testsuite-name", "relative",
		"--junitfile-testcase-classname", "relative",
		"--jsonfile", logFile,
		"--", goTestArgs...)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			gotestRetCode = exitError.ExitCode()
		} else {
			return fmt.Errorf("failed to run 'gotestsum': %w", err)
		}
	}
	fmt.Printf("Finished run, return code is %d\n", gotestRetCode)

	fmt.Printf("XML report written to %s\n", xmlFile)
	fmt.Printf("Test log (JSONL) written to %s\n", logFile)

	ansiLog := strings.ReplaceAll(logFile, ".jsonl", "-ansi.log")
	err = goRun("github.com/haveyoudebuggedit/gotestfmt/v2/cmd/gotestfmt@v2.3.1",
		"-input", logFile,
		"-showteststatus",
		"-nofail",
		">", ansiLog)
	if err != nil {
		return fmt.Errorf("failed to run 'gotestfmt': %w", err)
	}
	fmt.Printf("Test log (ANSI) written to %s\n", ansiLog)

	htmlLog := strings.ReplaceAll(logFile, ".jsonl", ".html")
	err = goRun("github.com/buildkite/terminal-to-html/v3/cmd/terminal-to-html@v3.6.1",
		"--preview", "<", ansiLog,
		">", htmlLog)
	if err != nil {
		return fmt.Errorf("failed to run 'terminal-to-html': %w", err)
	}
	fmt.Printf("Test log (HTML) written to %s\n", htmlLog)

	return nil
}

// Install Knative Serving in the current cluster.
// Parameters: crdsManifest - Knative Serving CRDs manifest.
//
//	coreManifest - Knative Serving core manifest.
//	netIstioManifest - Knative net-istio manifest.
func startKnativeServing(crdsManifest, coreManifest, netIstioManifest string) error {
	header("Starting Knative Serving")
	subheader("Installing Knative Serving")
	fmt.Printf("Installing Serving CRDs from %s\n", crdsManifest)
	err := kubectlApply(crdsManifest)
	if err != nil {
		return fmt.Errorf("failed to install Serving CRDs: %w", err)
	}

	fmt.Printf("Installing Serving core components from %s\n", coreManifest)
	err = kubectlApply(coreManifest)
	if err != nil {
		return fmt.Errorf("failed to install Serving core components: %w", err)
	}

	fmt.Printf("Installing net-istio components from %s\n", netIstioManifest)
	err = kubectlApply(netIstioManifest)
	if err != nil {
		return fmt.Errorf("failed to install net-istio components: %w", err)
	}

	err = waitUntilPodsRunning("knative-serving")
	if err != nil {
		return fmt.Errorf("timeout waiting for pods to be running: %w", err)
	}

	return nil
}

// Apply a Kubernetes manifest file using kubectl.
// Parameters: manifestFile - path to the manifest file.
func kubectlApply(manifestFile string) error {
	cmd := exec.Command("kubectl", "apply", "-f", manifestFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}
	return nil
}

// Get all pods in the given namespace.
// Parameters: namespace - the namespace to get pods from.
func kubectlGetPods(namespace string) ([]string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "--no-headers", "-n", namespace)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods failed: %w", err)
	}
	pods := strings.Split(strings.TrimSpace(string(output)), "\n")
	return pods, nil
}

// Filter out pods that are not in the "Running" state.
// Parameters: pods - the list of pods to filter.
func filterNotRunningPods(pods []string) []string {
	var notRunningPods []string
	for _, pod := range pods {
		if !strings.Contains(pod, "Running") {
			notRunningPods = append(notRunningPods, pod)
		}
	}
	return notRunningPods
}

// Install the stable release Knative/Serving in the current cluster.
// Parameters: version - Knative Serving version number, e.g., "0.6.0".
func startReleaseKnativeServing(version string) error {
	crdsManifest := fmt.Sprintf("https://storage.googleapis.com/knative-releases/serving/previous/v%s/serving-crds.yaml", version)
	coreManifest := fmt.Sprintf("https://storage.googleapis.com/knative-releases/serving/previous/v%s/serving-core.yaml", version)
	netIstioManifest := fmt.Sprintf("https://storage.googleapis.com/knative-releases/net-istio/previous/v%s/net-istio.yaml", version)

	return startKnativeServing(crdsManifest, coreManifest, netIstioManifest)
}

// Install the latest stable Knative/Serving in the current cluster.
func startLatestKnativeServing() error {
	crdsManifest := os.Getenv("KNATIVE_SERVING_RELEASE_CRDS")
	coreManifest := os.Getenv("KNATIVE_SERVING_RELEASE_CORE")
	netIstioManifest := os.Getenv("KNATIVE_NET_ISTIO_RELEASE")

	return startKnativeServing(crdsManifest, coreManifest, netIstioManifest)
}

// Install Knative Eventing extension in the current cluster.
func startKnativeEventingExtension(crdsManifest, namespace string) error {
	header("Starting Knative Eventing Extension")
	fmt.Printf("Installing Extension CRDs from %s\n", crdsManifest)
	kubectlApply(crdsManifest)
	err := waitUntilPodsRunning(namespace)
	if err != nil {
		return fmt.Errorf("failed to wait for pods in namespace %s to run: %w", namespace, err)
	}
	return nil
}

// Add function call to trap
// Parameters: f - Function to call
//
//	signals - Signals for trap
//
// helping:
func addTrap(f func(), signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	go func() {
		<-c
		f()
	}()
}

// Run a command, described by desc, for every go module in the project.
// Parameters:
//   - desc: Description of the command being run.
//   - args: Arguments to pass to the command.
func forEachGoModule(desc string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	modules, err := listGoModules()
	if err != nil {
		return err
	}

	for _, module := range modules {
		if err := os.Chdir(module); err != nil {
			return err
		}

		fmt.Printf("Running '%s' in module: %s\n", desc, module)

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Command '%s' failed in module %s: %v\n", desc, module, err)
			return err
		}

		if err := os.Chdir(".."); err != nil {
			return err
		}
	}

	return nil
}

// Helper function to list all Go modules in the project.
func listGoModules() ([]string, error) {
	output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return nil, err
	}

	modules := strings.Split(strings.TrimSpace(string(output)), "\n")
	return modules, nil
}

// Update Go dependencies.
// Parameters:
//   - upgrade: bool (flag), if set to true, perform an upgrade.
//   - release: string (flag), the release version to upgrade Knative components. Default is "main".
//   - moduleRelease: string (flag), a different go module tag for a release.
func goUpdateDeps(upgrade bool, release string, moduleRelease string) error {
	if upgrade {
		return forEachGoModule("Updating dependencies", "go", "get", "-u")
	}
	return nil
}

func updateModuleDeps(module string, release string, moduleRelease string, domain string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = module
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("go", "mod", "vendor")
	cmd.Dir = module
	if err := cmd.Run(); err != nil {
		return err
	}

	vendorDir := filepath.Join(module, "vendor")
	if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
		return nil
	}

	cmd = exec.Command("find", vendorDir, "-type", "f", "(",
		"-name", "OWNERS",
		"-o", "-name", "OWNERS_ALIASES",
		"-o", "-name", "BUILD",
		"-o", "-name", "BUILD.bazel",
		"-o", "-name", "*_test.go",
		")", "-exec", "rm", "-f", "{}", "+")
	if err := cmd.Run(); err != nil {
		return err
	}

	os.Setenv("GOFLAGS", "-mod=vendor")

	cmd = exec.Command("update_licenses", "third_party/VENDOR-LICENSE", "./...")
	cmd.Dir = module
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("remove_broken_symlinks", "./vendor")
	cmd.Dir = module
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// Return the go module name of the current module.
// Intended to be used like:
// 	moduleName := goModModuleName()
func goModModuleName() (string, error) {
	output, err := goRun("knative.dev/toolbox/modscope@latest current")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve module name: %w", err)
	}
	return string(output), nil
}

// Return a GOPATH to a temp directory. Works around the out-of-GOPATH issues
// for k8s client gen mixed with go mod.
// Intended to be used like:
//	gopath := goModGopathHack()
func goModGopathHack() string {
	// Skip this if the directory is already checked out onto the GOPATH.
	gopath := os.Getenv("GOPATH")
	if gopath != "" && filepath.HasPrefix(REPO_ROOT_DIR, gopath) {
		return gopath
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		fmt.Printf("Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}

	moduleName, err := goModModuleName()

	tmpRepoPath := filepath.Join(tmpDir, "src", )
	if err := os.MkdirAll(filepath.Dir(tmpRepoPath), 0755); err != nil {
		fmt.Printf("Failed to create parent directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.Symlink(REPO_ROOT_DIR, tmpRepoPath); err != nil {
		fmt.Printf("Failed to create symlink: %v\n", err)
		os.Exit(1)
	}

	return tmpDir
}

// updateLicenses runs go-licenses to update licenses.
// Parameters:
//   - outputFile: the output file path, relative to the repo root directory.
//   - directory: the directory to inspect.
func updateLicenses(outputFile, directory string) error {
	err := goRun("github.com/google/go-licenses@v1.2.1", "save", directory, "--save_path="+outputFile, "--force")
	if err != nil {
		return fmt.Errorf("failed to run go-licenses: %w", err)
	}
	return nil
}

// checkLicenses runs go-licenses to check for forbidden licenses.
func checkLicenses() error {
	err := goRun("github.com/google/go-licenses@v1.2.1", "check", REPO_ROOT_DIR+"/...")
	if err != nil {
		return fmt.Errorf("failed to run go-licenses: %w", err)
	}
	return nil
}

// isInt returns whether the given parameter is an integer.
func isInt(param string) bool {
	match, _ := regexp.MatchString(`^[0-9]+$`, param)
	return match
}

// isProtectedGCR returns whether the given parameter is the knative release/nightly GCR.
func isProtectedGCR(param string) bool {
	match, _ := regexp.MatchString(`^gcr.io/knative-(releases|nightly)/?$`, param)
	return match
}

// isProtectedCluster returns whether the given parameter is a protected cluster.
func isProtectedCluster(param string) bool {
	knativeTestsProject := os.Getenv("KNATIVE_TESTS_PROJECT")
	match, _ := regexp.MatchString(`^gke_`+knativeTestsProject+`_us\-[a-zA-Z0-9]+\-[a-z]+_[a-z0-9\-]+$`, param)
	return match
}

// isProtectedProject returns whether the given parameter is ${KNATIVE_TESTS_PROJECT}.
func isProtectedProject(param string) bool {
	knativeTestsProject := os.Getenv("KNATIVE_TESTS_PROJECT")
	return param == knativeTestsProject
}

// removeBrokenSymlinks removes symlinks in a path that are broken or lead outside the repo.
func removeBrokenSymlinks(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		linkPath := filepath.Join(path, file.Name())
		linkInfo, err := os.Lstat(linkPath)
		if err != nil {
			return fmt.Errorf("failed to get link info: %w", err)
		}

		if linkInfo.Mode()&os.ModeSymlink != os.ModeSymlink {
			continue
		}

		target, err := os.Readlink(linkPath)
		if err != nil {
			return fmt.Errorf("failed to read link target: %w", err)
		}

		// Remove broken symlinks
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("failed to remove broken symlink: %w", err)
			}
			continue
		}

		// Get canonical path to target, remove if outside the repo
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("failed to get absolute path to target: %w", err)
		}

		isProtectedRepo := isProtectedRepo(absTarget)
		if !isProtectedRepo {
			if err := os.Remove(linkPath); err != nil {
				return fmt.Errorf("failed to remove symlink outside the repo: %w", err)
			}
			continue
		}
	}

	return nil
}

// isProtectedRepo returns whether the given path is a protected repository.
// Modify this function according to your repository structure and rules.
func isProtectedRepo(path string) bool {
	// Check if the path is within the protected repositories
	// Modify this condition based on your repository structure and rules
	return filepath.HasPrefix(path, "/path/to/protected/repositories")
}

// getCanonicalPath returns the canonical path of a filesystem object.
func getCanonicalPath(path, baseDir string) (string, error) {
	if !filepath.IsAbs(path) {
		// Join the path with the base directory to get the absolute path
		path = filepath.Join(baseDir, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

// ListChangedFiles lists the changed files in the current PR.
func ListChangedFiles() ([]string, error) {
	var cmd *exec.Cmd

	if os.Getenv("PULL_BASE_SHA") != "" && os.Getenv("PULL_PULL_SHA") != "" {
		// Avoid warning when there are more than 1085 files renamed:
		// https://stackoverflow.com/questions/7830728/warning-on-diff-renamelimit-variable-when-doing-git-push
		cmd = exec.Command("git", "config", "diff.renames", "0")
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to configure git diff.renames: %w", err)
		}

		// Retrieve changed files using the specified commit range
		cmd = exec.Command("git", "--no-pager", "diff", "--name-only", os.Getenv("PULL_BASE_SHA")+".."+os.Getenv("PULL_PULL_SHA"))
	} else {
		// Do our best if not running in Prow, retrieve changed files with default commit range
		cmd = exec.Command("git", "diff", "--name-only", "HEAD^")
	}

	// Execute the git command and capture the output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list changed files: %w", err)
	}

	// Split the output into a slice of file names
	fileList := strings.Split(strings.TrimSpace(string(output)), "\n")
	return fileList, nil
}

// CurrentBranch returns the current branch.
func CurrentBranch() (string, error) {
	var branchName string

	// Get the branch name from Prow's env var
	if os.Getenv("IS_PROW") != "" {
		branchName = os.Getenv("PULL_BASE_REF")
	}

	// If Prow's env var is empty, try getting the current branch from other sources
	if branchName == "" {
		// Check if the GITHUB_BASE_REF env var is set
		branchName = os.Getenv("GITHUB_BASE_REF")
	}

	// If GITHUB_BASE_REF is empty, use git to get the current branch
	if branchName == "" {
		output, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		branchName = strings.TrimSpace(string(output))
	}

	return branchName, nil
}

// IsReleaseBranch returns whether the current branch is a release branch.
func IsReleaseBranch() bool {
	branchName, err := CurrentBranch()
	if err != nil {
		// Handle error if failed to get current branch
		return false
	}

	// Define the regular expression pattern for release branches
	re := regexp.MustCompile(`^release-[0-9\.]+$`)

	return re.MatchString(branchName)
}

// GetLatestKnativeYAMLSource returns the URL to the latest manifest for the given Knative project.
func GetLatestKnativeYAMLSource(repoName, yamlName string) (string, error) {
	if IsReleaseBranch() {
		branchName, err := CurrentBranch()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
		majorMinor := strings.TrimPrefix(branchName, "release-")
		yamlSourcePath, err := FindLatestReleaseManifest(repoName, majorMinor, yamlName)
		if err != nil {
			// Fall back to nightly
			return fmt.Sprintf("https://storage.googleapis.com/knative-nightly/%s/latest/%s.yaml", repoName, yamlName), nil
		}
		return fmt.Sprintf("https://storage.googleapis.com/%s", yamlSourcePath), nil
	}
	return fmt.Sprintf("https://storage.googleapis.com/knative-nightly/%s/latest/%s.yaml", repoName, yamlName), nil
}

// FindLatestReleaseManifest finds the latest release manifest with the given major&minor version.
func FindLatestReleaseManifest(repoName, majorMinor, yamlName string) (string, error) {
	cmd := exec.Command("gsutil", "ls", fmt.Sprintf("gs://knative-releases/%s/previous/v%s.*/%s.yaml", repoName, majorMinor, yamlName))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute gsutil command: %w", err)
	}
	manifestPaths := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(manifestPaths) == 0 {
		return "", fmt.Errorf("no release manifest found for %s v%s", repoName, majorMinor)
	}
	latestManifestPath := manifestPaths[len(manifestPaths)-1]
	return latestManifestPath, nil
}

func ShellcheckNewFiles() error {
	var arrayOfFiles []string
	failed := false

	shellcheckIgnoreFiles := os.Getenv("SHELLCHECK_IGNORE_FILES")
	if shellcheckIgnoreFiles == "" {
		shellcheckIgnoreFiles = "^vendor/"
	}

	output, err := ListChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to list changed files: %w", err)
	}

	arrayOfFiles = output
	for _, filename := range arrayOfFiles {
		if regexp.MustCompile(shellcheckIgnoreFiles).MatchString(filename) {
			continue
		}
		if isShellScript(filename) {
			if !runShellcheck(filename) {
				fmt.Printf("--- FAIL: shellcheck on %s\n", filename)
				failed = true
			} else {
				fmt.Printf("--- PASS: shellcheck on %s\n", filename)
			}
		}
	}

	if failed {
		return fmt.Errorf("shellcheck failures")
	}
	return nil
}

func isShellScript(filename string) bool {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return fileInfo.Mode().IsRegular() && strings.Contains(fileInfo.Mode().String(), "x")
}

func runShellcheck(filename string) bool {
	cmd := exec.Command("shellcheck", "-e", "SC1090", filename)
	err := cmd.Run()
	return err == nil
}

func latestVersion() string {
	branchName := CurrentBranch()

	// Use the latest release for main
	if branchName == "main" || branchName == "master" {
		cmd := exec.Command("git", "tag", "-l", "*v[0-9]*")
		output, err := cmd.Output()
		if err == nil {
			tags := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(tags) > 0 {
				latestTag := tags[0]
				splitTag := strings.Split(latestTag, "-")
				if len(splitTag) > 1 {
					return splitTag[1]
				}
			}
		}
		return ""
	}

	// Ideally we shouldn't need to treat release branches differently but
	// there are scenarios where git describe will return newer tags than
	// the ones on the current branch
	//
	// ie. create a PR pulling commits from 0.24 into a release-0.23 branch
	if strings.HasPrefix(branchName, "release-") {
		tag := strings.TrimPrefix(branchName, "release-")
		return tag
	}

	// Nearest tag with the `knative-` prefix
	cmd := exec.Command("git", "describe", "--abbrev=0", "--match", "knative-v[0-9]*")
	output, err := cmd.Output()
	if err == nil {
		tag := strings.TrimPrefix(strings.TrimSpace(string(output)), "knative-")
		return tag
	}

	// Fallback to older tag scheme vX.Y.Z
	cmd = exec.Command("git", "describe", "--abbrev=0", "--match", "v[0-9]*")
	output, err = cmd.Output()
	if err == nil {
		tag := strings.TrimPrefix(strings.TrimSpace(string(output)), "v")
		return tag
	}

	return ""
}

func getLatestPatchRelease(majorVersion, minorVersion string) string {
	var tagFilter string
	if majorVersion == "1" && minorVersion == "0" {
		tagFilter = "v0.26*"
	} else {
		minorInt, _ := strconv.Atoi(minorVersion)
		tagFilter = "v" + majorVersion + "." + strconv.Itoa(minorInt-1) + "*"
	}

	cmd := exec.Command("git", "tag", "-l", tagFilter)
	output, err := cmd.Output()
	if err == nil {
		tags := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(tags) > 0 {
			return tags[0]
		}
	}

	return ""
}