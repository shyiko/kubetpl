package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/shyiko/kubetpl/tpl"
	yamlext "github.com/shyiko/kubetpl/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var version string

func init() {
	log.SetFormatter(&simpleFormatter{})
	log.SetLevel(log.InfoLevel)
}

type simpleFormatter struct{}

func (f *simpleFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s ", entry.Message)
	for k, v := range entry.Data {
		fmt.Fprintf(b, "%s=%+v ", k, v)
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func main() {
	var format, chroot string
	var configFiles, configKeyValuePairs []string
	var chrootTemplateDir bool
	rootCmd := &cobra.Command{
		Use:  "kubetpl",
		Long: "Kubernetes templates made easy (https://github.com/shyiko/kubetpl).",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
				fmt.Println(version)
				return nil
			}
			return pflag.ErrHelp
		},
	}
	renderCmd := &cobra.Command{
		Use:     "render [file...]",
		Aliases: []string{"r"},
		Short:   "Render template(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return pflag.ErrHelp
			}
			config, err := readConfigFiles(configFiles...)
			if err != nil {
				log.Fatal(err)
			}
			for _, pair := range configKeyValuePairs {
				split := strings.SplitN(pair, "=", 2)
				if len(split) != 2 {
					log.Fatalf("Expected <key>=<value> pair, instead got %#v", pair)
				}
				config[split[0]] = split[1]
			}
			var formatSlice []string
			if format != "" {
				formatSlice = append(formatSlice, format)
			}
			if pre030format, _ := cmd.Flags().GetString("type"); pre030format != "" {
				formatSlice = append(formatSlice, pre030format)
			}
			// use-shorthand-* flags below are deprecated
			if set, _ := cmd.Flags().GetBool("use-shorthand-P"); set {
				formatSlice = append(formatSlice, "shell")
			}
			if set, _ := cmd.Flags().GetBool("use-shorthand-G"); set {
				formatSlice = append(formatSlice, "go-template")
			}
			if set, _ := cmd.Flags().GetBool("use-shorthand-T"); set {
				formatSlice = append(formatSlice, "template-kind")
			}
			if len(formatSlice) > 1 {
				log.Fatalf("-t/--type/-P/-G/-T/--format cannot be used simultaneously")
			}
			var explicitFormat string
			if len(formatSlice) == 1 {
				explicitFormat = formatSlice[0]
			}
			if explicitFormat == "placeholder" {
				log.Warnf("--type=placeholder was deprecated" +
					" (please use `--format=shell` or `# kubetpl:type:shell` directive instead)")
				explicitFormat = "sh"
			}
			out, err := render(args, config, explicitFormat, chroot, chrootTemplateDir)
			if err != nil {
				log.Fatal(err)
			}
			if output, _ := cmd.Flags().GetString("output"); output != "" && output != "-" {
				err := ioutil.WriteFile(output, out, 0600)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				os.Stdout.Write(out)
				fmt.Println()
			}
			return nil
		},
		Example: "  kubetpl render template.yml -i staging.env -s KEY=VALUE --format=shell\n\n" +
			"  # if template contains \"# kubetpl:format:shell\" --format can be omitted (recommended)\n" +
			"  kubetpl render template.yml -i staging.env -s KEY=VALUE",
	}
	renderCmd.Flags().StringP("type", "t", "", "Template flavor (shell|go-template|template-kind)")
	renderCmd.Flags().MarkDeprecated("type",
		"use --format=<shell|go-template|template-kind> instead\n"+
			"(if you wish to avoid typing --format=... - "+
			"add \"# kubetpl:format:<shell, go-template or template-kind>\" comment (preferably at the top of the template))")
	renderCmd.Flags().StringVar(&format, "format", "", "Template flavor (shell|go-template|template-kind)")
	renderCmd.Flags().BoolP("shorthand-P", "P", false, "")
	renderCmd.Flags().BoolP("shorthand-G", "G", false, "")
	renderCmd.Flags().BoolP("shorthand-T", "T", false, "")
	renderCmd.Flags().MarkDeprecated("shorthand-P",
		"use --format=shell instead\n"+
			"(if you wish to avoid typing --format=... - "+
			"add \"# kubetpl:format:shell\" comment (preferably at the top of the template))")
	renderCmd.Flags().MarkDeprecated("shorthand-G",
		"use --format=go-template instead\n"+
			"(if you wish to avoid typing --format=... - "+
			"add \"# kubetpl:format:go-template\" comment (preferably at the top of the template))")
	renderCmd.Flags().MarkDeprecated("shorthand-T",
		"use --format=template-kind instead\n"+
			"(if you wish to avoid typing --format=... - "+
			"add \"# kubetpl:format:template-kind\" comment (preferably at the top of the template))")
	renderCmd.Flags().StringArrayVarP(&configFiles, "input", "i", nil, "Config file(s) (*.{env,yml,yaml,json})")
	renderCmd.Flags().StringArrayVarP(&configKeyValuePairs, "set", "s", []string{},
		"<key>=<value> pair(s) (takes precedence over config file(s))")
	renderCmd.Flags().StringVarP(&chroot, "chroot", "c", "",
		"\"kubetpl/data-from-file\" root directory (access to anything outside of it will be denied)")
	renderCmd.Flags().BoolVarP(&chrootTemplateDir, "chroot-dirname", "C", false,
		"--chroot=<directory containing template> shorthand")
	renderCmd.Flags().StringP("output", "o", "", "Redirect output to a file")
	rootCmd.AddCommand(renderCmd)
	walk(rootCmd, func(cmd *cobra.Command) {
		cmd.Flags().BoolP("help", "h", false, "Print usage")
		cmd.Flags().MarkHidden("help")
	})
	rootCmd.PersistentFlags().Bool("debug", false, "Turn on debug output")
	rootCmd.Flags().Bool("version", false, "Print version information")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func walk(cmd *cobra.Command, cb func(*cobra.Command)) {
	cb(cmd)
	for _, c := range cmd.Commands() {
		walk(c, cb)
	}
}

func render(templateFiles []string, config map[string]interface{}, flavor string, chroot string, dirnameChroot bool) ([]byte, error) {
	if chroot != "" {
		var err error
		chroot, err = filepath.Abs(chroot)
		if err != nil {
			return nil, err
		}
	}
	var objs []map[interface{}]interface{}
	for _, templateFile := range templateFiles {
		t, err := newTemplate(templateFile, flavor)
		if err != nil {
			return nil, err
		}
		out, err := t.Render(config)
		if err != nil {
			return nil, err
		}
		baseDir, err := dirnameAbs(templateFile)
		if err != nil {
			return nil, err
		}
		templateChroot := chroot
		if chroot == "" && dirnameChroot {
			templateChroot = baseDir
		}
		if !strings.HasSuffix(templateChroot, string(filepath.Separator)) {
			templateChroot += string(filepath.Separator)
		}
		for _, chunk := range yamlext.Chunk(out) {
			obj := make(map[interface{}]interface{})
			if err = yaml.Unmarshal(chunk, &obj); err != nil {
				return nil, err
			}
			if _, err := tpl.ReplaceDataFromFileInPlace(obj, func(file string) (string, []byte, error) {
				if !filepath.IsAbs(file) {
					file = filepath.Join(baseDir, file)
				}
				file, err := filepath.Abs(file)
				if err != nil {
					return "", nil, err
				}
				if templateChroot == "" || !strings.HasPrefix(file, templateChroot) {
					return "", nil, fmt.Errorf("%s is denied access to %s"+
						" (use -c/--chroot=<root dir> or -C/--chroot-dirname (--chroot=<directory containing template> shorthand) to allow)",
						templateFile, file)
				}
				data, err := ioutil.ReadFile(file)
				return filepath.Base(file), data, err
			}); err != nil {
				return nil, err
			}
			objs = append(objs, obj)
		}
	}
	var buf bytes.Buffer
	for _, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		buf.Write([]byte("---\n"))
		buf.Write(o)
	}
	return buf.Bytes(), nil
}

func dirnameAbs(path string) (string, error) {
	if path == "-" {
		return os.Getwd()
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", fmt.Errorf("\"kubetpl/data-from-file\" + https?:// is not supported at the moment")
	}
	return filepath.Abs(filepath.Dir(path))
}

func newTemplate(file string, flavor string) (tpl.Template, error) {
	content, err := readFile(file)
	if err != nil {
		return nil, err
	}
	directives, err := parseDirectives(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", file, err.Error())
	}
	for _, d := range directives {
		if d.Key == "format" {
			flavor = d.Value
		}
	}
	if flavor == "" {
		if hasExtensionAny(file, ".yml.kubetpl", ".yaml.kubetpl", ".json.kubetpl-go") {
			log.Warnf("*.{yml,yaml,json}.kubetpl as an indicator for \"shell\" flavor has been deprecated" +
				" (please use `--format=shell` or `# kubetpl:type:shell` directive instead)")
			flavor = "shell"
		}
		if hasExtensionAny(file, ".yml.kubetpl-go", ".yaml.kubetpl-go", ".json.kubetpl-go") {
			log.Warnf("*.{yml,yaml,json}.kubetpl-go as an indicator for \"go-template\" flavor has been deprecated" +
				" (please use `--format=go-template` or `# kubetpl:format:go-template` directive instead)")
			flavor = "go-template"
		}
	}
	switch flavor {
	case "sh", "shell":
		return tpl.NewShellTemplate(content)
	case "go", "go-template":
		return tpl.NewGoTemplate(content, file)
	case "template-kind":
		return tpl.NewTemplateKindTemplate(content)
	default:
		if flavor != "" {
			return nil, fmt.Errorf("%s: unknown template type \"%s\" "+
				"(expected \"shell\", \"go-template\" or \"template-kind\")", file, flavor)
		}
		// warn if "kind: Template" is present
		for _, chunk := range yamlext.Chunk(content) {
			m := make(map[interface{}]interface{})
			if err = yaml.Unmarshal(chunk, &m); err != nil {
				return nil, fmt.Errorf("%s does not appear to be a valid YAML.\n"+
					"Did you forget to specify `--format=<shell|go-template|template-kind>` / add \"# kubetpl:format:<shell|go-template|template-kind>\"?", file)
			}
			if m["kind"] == "Template" {
				log.Warnf("%s appears to contain \"kind: Template\"" +
					" (please either use `--format=go-template` or add \"# kubetpl:format:go-template\" to the template)")
				break
			}
		}
		return tpl.NewTemplateKindTemplate(content) // change to simple pass-through in 1.0.0
	}
}

func readConfigFiles(path ...string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	for _, path := range path {
		cfg, err := readConfigFile(path)
		if err != nil {
			return nil, err
		}
		for key, value := range cfg {
			config[key] = value
		}
	}
	return config, nil
}

func readConfigFile(path string) (map[string]interface{}, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if hasExtension(path, ".env") {
		return parseDotEnv(data)
	}
	return parseYAML(data)
}

func hasExtensionAny(path string, ext ...string) bool {
	for _, suffix := range ext {
		if hasExtension(path, suffix) {
			return true
		}
	}
	return false
}

func hasExtension(path string, ext string) bool {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		path = strings.SplitN(path, "#", 2)[0]
		path = strings.SplitN(path, "?", 2)[0]
	}
	return strings.HasSuffix(path, ext)
}

func readFile(path string) ([]byte, error) {
	if path == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		res, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return nil, fmt.Errorf(`GET "%s" %d`, path, res.StatusCode)
		}
		return ioutil.ReadAll(res.Body)
	}
	return ioutil.ReadFile(path)
}

func parseYAML(data []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	yaml.Unmarshal(data, &m)
	return m, nil
}

func parseDotEnv(data []byte) (map[string]interface{}, error) {
	f, err := ini.Load(data)
	if err != nil {
		return nil, err
	}
	section, err := f.GetSection("")
	if err != nil {
		panic(err)
	}
	m := map[string]interface{}{}
	for key, value := range section.KeysHash() {
		m[key] = value
	}
	return m, nil
}

type directive struct {
	Key, Value string
}

func parseDirectives(s []byte) ([]directive, error) {
	var d []directive
	for _, line := range strings.Split(string(s), "\n") {
		if strings.HasPrefix(line, "# kubetpl:") {
			split := append(strings.SplitN(line[strings.Index(line, ":")+1:], ":", 2), "")
			key, value := strings.ToLower(split[0]), strings.ToLower(split[1])
			if key != "format" {
				return nil, fmt.Errorf("unrecognized # kubetpl:%s directive", key)
			}
			d = append(d, directive{key, value})
		}
	}
	return d, nil
}
