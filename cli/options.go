package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type GlobalOptions struct {
	Verbose        bool
	Debug          bool
	NonInteractive bool
	File           string
	Namespace      string
	Selector       string
	TemplateDirs   []string
	ParamDirs      []string
	PublicKeyDir   string
	PrivateKey     string
	Passphrase     string
	IsLoggedIn     bool
}

type CompareOptions struct {
	*GlobalOptions
	Labels                  string
	Params                  []string
	ParamFiles              []string
	Diff                    string
	IgnorePaths             []string
	IgnoreUnknownParameters bool
	UpsertOnly              bool
	Force                   bool
	Resource                string
}

type ExportOptions struct {
	*GlobalOptions
	Resource string
}

func GetFileFlags(filename string, verboseOrDebug bool) (map[string]string, error) {
	fileFlags := make(map[string]string)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if filename == "Tailorfile" {
			if verboseOrDebug {
				PrintBluef("--> No file '%s' found.\n", filename)
			}
			return fileFlags, nil
		}
		return fileFlags, err
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return fileFlags, err
	}
	content := string(b)
	text := strings.TrimSuffix(content, "\n")
	lines := strings.Split(text, "\n")

	for _, untrimmedLine := range lines {
		line := strings.TrimSpace(untrimmedLine)
		if len(line) == 0 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		pair := strings.SplitN(line, " ", 2)
		if len(pair) == 2 {
			key := pair[0]
			value := strings.TrimSpace(pair[1])
			if val, ok := fileFlags[key]; ok {
				value = val + "," + value
			}
			fileFlags[key] = value
		} else {
			fileFlags["resource"] = pair[0]
		}
	}
	return fileFlags, nil
}

func (o *GlobalOptions) UpdateWithFile(fileFlags map[string]string) {
	if fileFlags["verbose"] == "true" {
		o.Verbose = true
	}
	if fileFlags["debug"] == "true" {
		o.Debug = true
	}
	if fileFlags["non-interactive"] == "true" {
		o.NonInteractive = true
	}
	if val, ok := fileFlags["namespace"]; ok {
		o.Namespace = val
	}
	if val, ok := fileFlags["selector"]; ok {
		o.Selector = val
	}
	if val, ok := fileFlags["template-dir"]; ok {
		o.TemplateDirs = strings.Split(val, ",")
	}
	if val, ok := fileFlags["param-dir"]; ok {
		o.ParamDirs = strings.Split(val, ",")
	}
	if val, ok := fileFlags["public-key-dir"]; ok {
		o.PublicKeyDir = val
	}
	if val, ok := fileFlags["private-key"]; ok {
		o.PrivateKey = val
	}
	if val, ok := fileFlags["passphrase"]; ok {
		o.Passphrase = val
	}
}

func (o *GlobalOptions) UpdateWithFlags(verboseFlag bool, debugFlag bool, nonInteractiveFlag bool, namespaceFlag string, selectorFlag string, templateDirFlag []string, paramDirFlag []string, publicKeyDirFlag string, privateKeyFlag string, passphraseFlag string) {
	if verboseFlag {
		o.Verbose = true
	}

	if debugFlag {
		o.Debug = true
	}

	if nonInteractiveFlag {
		o.NonInteractive = true
	}

	if len(namespaceFlag) > 0 {
		o.Namespace = namespaceFlag
	}

	if len(selectorFlag) > 0 {
		o.Selector = selectorFlag
	}

	if len(o.TemplateDirs) == 0 {
		o.TemplateDirs = templateDirFlag
	} else if len(templateDirFlag) > 1 || templateDirFlag[0] != "." {
		o.TemplateDirs = templateDirFlag
	}

	if len(o.ParamDirs) == 0 {
		o.ParamDirs = paramDirFlag
	} else if len(paramDirFlag) > 1 || paramDirFlag[0] != "." {
		o.ParamDirs = paramDirFlag
	}

	if len(o.PublicKeyDir) == 0 || publicKeyDirFlag != "." {
		o.PublicKeyDir = publicKeyDirFlag
	}

	if len(o.PrivateKey) == 0 || privateKeyFlag != "private.key" {
		o.PrivateKey = privateKeyFlag
	}

	if len(passphraseFlag) > 0 {
		o.Passphrase = passphraseFlag
	}
}

func (o *GlobalOptions) Process() error {
	verbose = o.Verbose || o.Debug
	debug = o.Debug
	if len(o.Namespace) == 0 {
		n, err := getOcNamespace()
		if err != nil {
			return err
		}
		o.Namespace = n
	} else {
		if !o.CheckLoggedIn() {
			return errors.New("You need to login with 'oc login' first.")
		}
		err := checkOcNamespace(o.Namespace)
		if err != nil {
			return fmt.Errorf("No such project: %s", o.Namespace)
		}
	}
	return nil
}

func (o *GlobalOptions) CheckLoggedIn() bool {
	if !o.IsLoggedIn {
		cmd := exec.Command("oc", "whoami")
		_, err := cmd.CombinedOutput()
		o.IsLoggedIn = (err == nil)
	}
	return o.IsLoggedIn
}

func (o *CompareOptions) UpdateWithFile(fileFlags map[string]string) {
	if val, ok := fileFlags["labels"]; ok {
		o.Labels = val
	}
	if val, ok := fileFlags["param"]; ok {
		o.Params = strings.Split(val, ",")
	}
	if val, ok := fileFlags["param-file"]; ok {
		o.ParamFiles = strings.Split(val, ",")
	}
	if val, ok := fileFlags["diff"]; ok {
		o.Diff = val
	}
	if fileFlags["ignore-unknown-parameters"] == "true" {
		o.IgnoreUnknownParameters = true
	}
	if fileFlags["upsert-only"] == "true" {
		o.UpsertOnly = true
	}
	if fileFlags["force"] == "true" {
		o.Force = true
	}
	if val, ok := fileFlags["ignore-path"]; ok {
		o.IgnorePaths = strings.Split(val, ",")
	}
	if val, ok := fileFlags["resource"]; ok {
		o.Resource = val
	}
}

func (o *CompareOptions) UpdateWithFlags(labelsFlag string, paramFlag []string, paramFileFlag []string, diffFlag string, ignorePathFlag []string, ignoreUnknownParametersFlag bool, upsertOnlyFlag bool, forceFlag bool, resourceArg string) {
	if len(labelsFlag) > 0 {
		o.Labels = labelsFlag
	}
	// Update / override params
	if len(paramFlag) > 0 {
		params := map[string]string{}
		for _, setParam := range o.Params {
			setPair := strings.SplitN(setParam, "=", 2)
			key := setPair[0]
			params[key] = setPair[1]
			for _, newParam := range paramFlag {
				newPair := strings.SplitN(newParam, "=", 2)
				if key == newPair[0] {
					params[key] = newPair[1]
					break
				}
			}
		}
		o.Params = []string{}
		for k, v := range params {
			o.Params = append(o.Params, k+"="+v)
		}
		for _, v := range paramFlag {
			pair := strings.SplitN(v, "=", 2)
			if _, ok := params[pair[0]]; !ok {
				o.Params = append(o.Params, v)
			}
		}
	}
	if len(paramFileFlag) > 0 {
		o.ParamFiles = paramFileFlag
	}
	if len(diffFlag) > 0 {
		o.Diff = diffFlag
	}
	if ignoreUnknownParametersFlag {
		o.IgnoreUnknownParameters = true
	}
	if upsertOnlyFlag {
		o.UpsertOnly = true
	}
	if forceFlag {
		o.Force = true
	}
	if len(ignorePathFlag) > 0 {
		o.IgnorePaths = ignorePathFlag
	}
	if len(resourceArg) > 0 {
		o.Resource = resourceArg
	}
}

func (o *CompareOptions) Process() error {
	if (len(o.ParamDirs) > 1 || o.ParamDirs[0] != ".") && len(o.ParamFiles) > 0 {
		return errors.New("You cannot specify both --param-dir and --param-file")
	}
	for _, p := range o.ParamDirs {
		if p != "." {
			if _, err := os.Stat(p); os.IsNotExist(err) {
				return fmt.Errorf("Param directory %s does not exist", p)
			}
		}
	}
	if o.Diff != "text" && o.Diff != "json" {
		return errors.New("--diff must be either text or json")
	}
	if !o.CheckLoggedIn() {
		return errors.New("You need to login with 'oc login' first")
	}
	if strings.Contains(o.Resource, "/") && len(o.Selector) > 0 {
		DebugMsg("Ignoring selector", o.Selector, "as resource is given")
		o.Selector = ""
	}
	return nil
}

func (o *ExportOptions) UpdateWithFile(fileFlags map[string]string) {
	if val, ok := fileFlags["resource"]; ok {
		o.Resource = val
	}
}

func (o *ExportOptions) UpdateWithFlags(resourceArg string) {
	if len(resourceArg) > 0 {
		o.Resource = resourceArg
	}
}

func (o *ExportOptions) Process() error {
	if !o.CheckLoggedIn() {
		return errors.New("You need to login with 'oc login' first")
	}
	if strings.Contains(o.Resource, "/") && len(o.Selector) > 0 {
		DebugMsg("Ignoring selector", o.Selector, "as resource is given")
		o.Selector = ""
	}
	return nil
}

func getOcNamespace() (string, error) {
	cmd := ExecPlainOcCmd([]string{"project", "--short"})
	n, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(n)), err
}

func checkOcNamespace(n string) error {
	cmd := ExecPlainOcCmd([]string{"project", n, "--short"})
	_, err := cmd.CombinedOutput()
	return err
}
