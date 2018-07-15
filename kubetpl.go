package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/shyiko/kubetpl/cli"
	"github.com/shyiko/kubetpl/dotenv"
	"github.com/shyiko/kubetpl/engine"
	"github.com/shyiko/kubetpl/engine/processor"
	yamlext "github.com/shyiko/kubetpl/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	completion := cli.NewCompletion()
	completed, err := completion.Execute()
	if err != nil {
		log.Debug(err)
		os.Exit(3)
	}
	if completed {
		os.Exit(0)
	}
	var syntax, chroot string
	var configFiles, configKeyValuePairs, freezeRefs, freezeList []string
	var allowFsAccess, freeze bool
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
			if syntax != "" {
				formatSlice = append(formatSlice, syntax)
			}
			if pre030format, _ := cmd.Flags().GetString("type"); pre030format != "" {
				formatSlice = append(formatSlice, pre030format)
			}
			// shorthand-* flags below are deprecated
			if set, _ := cmd.Flags().GetBool("shorthand-P"); set {
				formatSlice = append(formatSlice, "$")
			}
			if set, _ := cmd.Flags().GetBool("shorthand-G"); set {
				formatSlice = append(formatSlice, "go-template")
			}
			if set, _ := cmd.Flags().GetBool("shorthand-T"); set {
				formatSlice = append(formatSlice, "template-kind")
			}
			if len(formatSlice) > 1 {
				log.Fatalf("-t/--type/-P/-G/-T/--syntax cannot be used simultaneously")
			}
			var explicitFormat string
			if len(formatSlice) == 1 {
				explicitFormat = formatSlice[0]
			}
			if explicitFormat == "placeholder" {
				log.Warnf("--type=placeholder was deprecated" +
					" (please use `--syntax=$` or `# kubetpl:type:$` directive instead)")
				explicitFormat = "$"
			}
			var normalizedFreezeList []string
			for _, v := range freezeList {
				ref, err := normalizeRef(v)
				if err != nil {
					return err
				}
				normalizedFreezeList = append(normalizedFreezeList, ref)
			}
			out, err := render(args, config, renderOpts{
				format:            explicitFormat,
				chroot:            chroot,
				chrootTemplateDir: allowFsAccess,
				freeze:            freeze,
				freezeRefs:        freezeRefs,
				freezeList:        normalizedFreezeList,
			})
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
		Example: "  kubetpl render template.yml -i staging.env -s KEY=VALUE --syntax=$\n\n" +
			"  # if template contains \"# kubetpl:syntax:<template flavor, e.g. $>\" --syntax can be omitted (recommended)\n" +
			"  kubetpl render template.yml -i staging.env -s KEY=VALUE",
	}
	renderCmd.Flags().BoolVarP(&freeze, "freeze", "z", false, "Freeze ConfigMap/Secret|s")
	renderCmd.Flags().StringArrayVar(&freezeRefs, "freeze-ref", nil,
		"External ConfigMap/Secret|s that should not be included in the output and yet references to which need to be '--freeze'd")
	renderCmd.Flags().StringSliceVar(&freezeList, "freeze-list", nil,
		"<kind>/<name>s to freeze (e.g. ConfigMap/foo, Secret/bar)")
	renderCmd.Flags().StringP("type", "t", "", "Template flavor ($, go-template or template-kind)")
	renderCmd.Flags().MarkDeprecated("type",
		"use --syntax=<$|go-template|template-kind> instead\n"+
			"(if you wish to avoid typing --syntax=... - "+
			"add \"# kubetpl:syntax:<$, go-template or template-kind>\" comment (preferably at the top of the template))")
	renderCmd.Flags().StringVarP(&syntax, "syntax", "x", "", "Template flavor ($, go-template or template-kind) (https://github.com/shyiko/kubetpl#template-flavors)")
	renderCmd.Flags().BoolP("shorthand-P", "P", false, "")
	renderCmd.Flags().BoolP("shorthand-G", "G", false, "")
	renderCmd.Flags().BoolP("shorthand-T", "T", false, "")
	renderCmd.Flags().MarkDeprecated("shorthand-P",
		"use --syntax=$ instead\n"+
			"(if you wish to avoid typing --syntax=... - "+
			"add \"# kubetpl:syntax:$\" comment (preferably at the top of the template))")
	renderCmd.Flags().MarkDeprecated("shorthand-G",
		"use --syntax=go-template instead\n"+
			"(if you wish to avoid typing --syntax=... - "+
			"add \"# kubetpl:syntax:go-template\" comment (preferably at the top of the template))")
	renderCmd.Flags().MarkDeprecated("shorthand-T",
		"use --syntax=template-kind instead\n"+
			"(if you wish to avoid typing --syntax=... - "+
			"add \"# kubetpl:syntax:template-kind\" comment (preferably at the top of the template))")
	renderCmd.Flags().StringArrayVarP(&configFiles, "input", "i", nil, "Config file(s) (*.{env,yml,yaml,json})")
	renderCmd.Flags().StringArrayVarP(&configKeyValuePairs, "set", "s", []string{},
		"<key>=<value> pairs (take precedence over --input files (if any))")
	renderCmd.Flags().StringVarP(&chroot, "chroot", "c", "",
		"The root directory in which extensions like \"kubetpl/data-from-file\" are to be allowed to read files\n"+
			"(access to anything outside of --chroot will denied)")
	renderCmd.Flags().BoolVar(&allowFsAccess, "allow-fs-access", false,
		`Shorthand for --chroot=<directory containing template>`)
	renderCmd.Flags().StringP("output", "o", "", "Redirect output to a file")
	rootCmd.AddCommand(renderCmd)
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Command-line completion",
	}
	completionCmd.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generate Bash completion",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 0 {
					return pflag.ErrHelp
				}
				if err := completion.GenBashCompletion(os.Stdout); err != nil {
					log.Error(err)
				}
				return nil
			},
			Example: "  source <(kubetpl completion bash)",
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generate Z shell completion",
			RunE: func(cmd *cobra.Command, args []string) error {
				if len(args) != 0 {
					return pflag.ErrHelp
				}
				if err := completion.GenZshCompletion(os.Stdout); err != nil {
					log.Error(err)
				}
				return nil
			},
			Example: "  source <(kubetpl completion zsh)",
		},
	)
	rootCmd.AddCommand(completionCmd)
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

func normalizeRef(v string) (string, error) {
	split := strings.SplitN(v, "/", 2)
	if len(split) != 2 {
		return "", fmt.Errorf(`"%s" is not a valid resource reference (expected <kind>/<name>, e.g. ConfigMap/app)`, v)
	}
	kind, name := strings.ToLower(split[0]), split[1]
	switch kind {
	case "cm", "configmap", "configmaps":
		kind = "ConfigMap"
	case "secret", "secrets":
		kind = "Secret"
	default:
		return "", fmt.Errorf(`Invalid kind "%s" (valid (case-insensitive): cm/configmaps/configmap, secrets/secret)`, v)
	}
	return kind + "/" + name, nil
}

func walk(cmd *cobra.Command, cb func(*cobra.Command)) {
	cb(cmd)
	for _, c := range cmd.Commands() {
		walk(c, cb)
	}
}

type renderOpts struct {
	format            string
	chroot            string
	chrootTemplateDir bool
	freeze            bool
	freezeRefs        []string
	freezeList        []string
}

func render(templateFiles []string, data map[string]interface{}, opts renderOpts) ([]byte, error) {
	objs, err := renderTemplates(templateFiles, data, opts)
	if err != nil {
		return nil, err
	}
	if opts.freeze || len(opts.freezeRefs) > 0 || len(opts.freezeList) > 0 {
		refs, err := renderTemplates(opts.freezeRefs, data, opts)
		if err != nil {
			return nil, err
		}
		if err := processor.FreezeInPlace(processor.FreezeRequest{
			Docs:    bodySlice(objs),
			Refs:    bodySlice(refs),
			Include: opts.freezeList,
		}); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	for _, obj := range objs {
		if len(obj.body) == 0 {
			continue
		}
		o, err := yaml.Marshal(obj.body)
		if err != nil {
			return nil, err
		}
		buf.Write([]byte("---\n"))
		buf.Write(o)
		if len(obj.footer) > 0 {
			// support for comment-less meta is planned in kubesec@1.0.0
			// until then "# kubesec:" footer is preserved to allow freezing
			for i, line := range bytes.Split(obj.footer, []byte("\n")) {
				if bytes.HasPrefix(line, []byte("# kubesec:")) {
					if i > 0 {
						buf.Write([]byte("\n"))
					}
					buf.Write(line)
				}
			}
		}
	}
	return buf.Bytes(), nil
}

type document struct {
	header []byte
	body   map[interface{}]interface{}
	footer []byte
}

func bodySlice(docs []document) []map[interface{}]interface{} {
	var r []map[interface{}]interface{}
	for _, doc := range docs {
		r = append(r, doc.body)
	}
	return r
}

func renderTemplates(templateFiles []string, config map[string]interface{}, opts renderOpts) ([]document, error) {
	chroot := opts.chroot
	if chroot != "" {
		var err error
		chroot, err = filepath.Abs(chroot)
		if err != nil {
			return nil, err
		}
	}
	var objs []document
	for _, templateFile := range templateFiles {
		docs, err := renderTemplate(templateFile, config, opts.format, chroot, opts.chrootTemplateDir)
		if err != nil {
			return nil, err
		}
		objs = append(objs, docs...)
	}
	return objs, nil
}

func renderTemplate(templateFile string, config map[string]interface{}, format string, chroot string, chrootTemplateDir bool) ([]document, error) {
	t, err := newTemplate(templateFile, format)
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
	if chroot == "" && chrootTemplateDir {
		templateChroot = baseDir
	}
	if templateChroot != "" && !strings.HasSuffix(templateChroot, string(filepath.Separator)) {
		templateChroot += string(filepath.Separator)
	}
	var objs []document
	for _, chunk := range yamlext.Chunk(out) {
		obj := make(map[interface{}]interface{})
		if err = yaml.Unmarshal(chunk, &obj); err != nil {
			return nil, err
		}
		if _, err := processor.ReplaceDataFromFileInPlace(obj, func(path string) (string, []byte, error) {
			file := path
			if !filepath.IsAbs(file) {
				file = filepath.Join(baseDir, file)
			}
			file, err := filepath.Abs(file)
			if err != nil {
				return "", nil, err
			}
			if templateChroot == "" || !strings.HasPrefix(file, templateChroot) {
				fileRel := file
				if cwd, err := os.Getwd(); err == nil {
					if p, err := filepath.Rel(cwd, file); err == nil {
						fileRel = p
					}
				}
				return "", nil, fmt.Errorf(`%s: access denied: %s`+
					" (use --allow-fs-access and/or -c/--chroot=<root dir, e.g. '.'> to allow)",
					templateFile, fileRel)
			}
			data, err := ioutil.ReadFile(file)
			return filepath.Base(file), data, err
		}); err != nil {
			return nil, err
		}
		objs = append(objs, document{
			header: yamlext.Header(chunk),
			body:   obj,
			footer: yamlext.Footer(chunk),
		})
	}
	return objs, nil
}

func dirnameAbs(path string) (string, error) {
	if path == "-" {
		return os.Getwd()
	}
	return filepath.Abs(filepath.Dir(path))
}

func newTemplate(file string, flavor string) (engine.Template, error) {
	content, err := readFile(file)
	if err != nil {
		return nil, err
	}
	content = bytes.Replace(content, []byte("\r\n"), []byte("\n"), -1)
	directives, err := parseDirectives(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", file, err.Error())
	}
	for _, d := range directives {
		if d.Key == "syntax" {
			flavor = d.Value
		}
	}
	if flavor == "" {
		if hasExtensionAny(file, ".yml.kubetpl", ".yaml.kubetpl", ".json.kubetpl-go") {
			log.Warnf("*.{yml,yaml,json}.kubetpl as an indicator for \"$\" flavor has been deprecated" +
				" (please use `--syntax=$` or `# kubetpl:syntax:$` directive instead)")
			flavor = "$"
		}
		if hasExtensionAny(file, ".yml.kubetpl-go", ".yaml.kubetpl-go", ".json.kubetpl-go") {
			log.Warnf("*.{yml,yaml,json}.kubetpl-go as an indicator for \"go-template\" flavor has been deprecated" +
				" (please use `--syntax=go-template` or `# kubetpl:syntax:go-template` directive instead)")
			flavor = "go-template"
		}
	}
	switch flavor {
	case "$":
		return engine.NewShellTemplate(content)
	case "go-template":
		return engine.NewGoTemplate(content, file)
	case "template-kind":
		return engine.NewTemplateKindTemplate(content, engine.TemplateKindTemplateDropNull())
	default:
		if flavor != "" {
			return nil, fmt.Errorf("%s: unknown template type \"%s\" "+
				"(expected \"$\", \"go-template\" or \"template-kind\")", file, flavor)
		}
		// warn if "kind: Template" is present
		for _, chunk := range yamlext.Chunk(content) {
			m := make(map[interface{}]interface{})
			if err = yaml.Unmarshal(chunk, &m); err != nil {
				return nil, fmt.Errorf("%s does not appear to be a valid YAML (%s).\n"+
					"Did you forget to specify `--syntax=<$|go-template|template-kind>`"+
					" / add \"# kubetpl:syntax:<$|go-template|template-kind>\"?", file, err.Error())
			}
			if m["kind"] == "Template" {
				log.Warnf("%s appears to contain \"kind: Template\"" +
					" (please either use `--syntax=go-template` or add \"# kubetpl:syntax:go-template\" to the template)")
				break
			}
		}
		return engine.NewTemplateKindTemplate(content) // change to simple pass-through in 1.0.0
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
	env, err := dotenv.Parse(data)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	for key, value := range env {
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
			if key != "syntax" {
				return nil, fmt.Errorf("unrecognized # kubetpl:%s directive", key)
			}
			d = append(d, directive{key, value})
		}
	}
	return d, nil
}
