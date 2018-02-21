package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/shyiko/kubetpl/tpl"
	yamlext "github.com/shyiko/kubetpl/yml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
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
	var format string
	var configFiles, configKeyValuePairs []string
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
		Use:     "render [file]",
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
			formatSlice := []string{}
			if format != "" {
				formatSlice = append(formatSlice, format)
			}
			if set, _ := cmd.Flags().GetBool("use-shorthand-P"); set {
				formatSlice = append(formatSlice, "placeholder")
			}
			if set, _ := cmd.Flags().GetBool("use-shorthand-G"); set {
				formatSlice = append(formatSlice, "go-template")
			}
			if set, _ := cmd.Flags().GetBool("use-shorthand-T"); set {
				formatSlice = append(formatSlice, "template-kind")
			}
			if len(formatSlice) > 1 {
				log.Fatalf("--type/-P/-G/-T cannot be used simultaneously")
			}
			var explicitFormat string
			if len(formatSlice) == 1 {
				explicitFormat = formatSlice[0]
			}
			out, err := render(args, config, explicitFormat)
			if err != nil {
				log.Fatal(err)
			}
			if output, _ := cmd.Flags().GetString("output"); output != "" && output != "-" {
				err := ioutil.WriteFile(output, []byte(out), 0600)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println(out)
			}
			return nil
		},
		Example: "  # render \"placeholder\" (aka $VAR / ${VAR}) type of template\n" +
			"  kubetpl render svc-and-deploy.yml.kubetpl -i staging.env -s KEY=VALUE\n" +
			"  # same as above\n" +
			"  kubetpl render svc-and-deploy.yml --type=placeholder -i staging.env -s KEY=VALUE\n" +
			"  # -P is a shorthand for --type=placeholder\n" +
			"  kubetpl render svc-and-deploy.yml -P -i staging.env -s KEY=VALUE\n" +
			"\n" +
			"  # render \"go-template\" type of template\n" +
			"  kubetpl render svc-and-deploy.yml.kubetpl-go -i staging.yml -s KEY=VALUE\n" +
			"  # same as above\n" +
			"  kubetpl render svc-and-deploy.yml --type=go-template -i staging.yml -s KEY=VALUE\n" +
			"  # -G is a shorthand for --type=go-template\n" +
			"  kubetpl render svc-and-deploy.yml -G -i staging.yml -s KEY=VALUE\n" +
			"\n" +
			"  # render \"template-kind\" (aka \"kind: Template\") type of template\n" +
			"  kubetpl render svc-and-deploy.yml -i staging.yml -s KEY=VALUE\n" +
			"  # same as above\n" +
			"  kubetpl render svc-and-deploy.yml --type=template-kind -i staging.yml -s KEY=VALUE\n" +
			"  # -T is a shorthand for --type=template-kind\n" +
			"  kubetpl render svc-and-deploy.yml -T -i staging.yml -s KEY=VALUE",
	}
	renderCmd.Flags().StringVarP(&format, "type", "t", "",
		"Template format\n\n    \"placeholder\" (*.{yml,yaml,json}.kubetpl), "+
			"\n    \"go-template\" (*.{yml,yaml,json}.kubetpl-go), \n    \"template-kind\" (*.{yml,yaml,json})")
	renderCmd.Flags().BoolP("use-shorthand-G", "G", false, "")
	renderCmd.Flags().BoolP("use-shorthand-T", "T", false, "")
	renderCmd.Flags().BoolP("use-shorthand-P", "P", false, "")
	renderCmd.Flags().MarkHidden("use-shorthand-G")
	renderCmd.Flags().MarkHidden("use-shorthand-T")
	renderCmd.Flags().MarkHidden("use-shorthand-P")
	renderCmd.Flags().StringArrayVarP(&configFiles, "input", "i", nil, "Config (data) file(s) (*.{env,yml,yaml,json})")
	renderCmd.Flags().StringArrayVarP(&configKeyValuePairs, "set", "s", []string{}, "<key>=<value> pair (takes precedence over data (config) file(s))")
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

func render(templateFiles []string, config map[string]interface{}, format string) (string, error) {
	var output []string
	for _, templateFile := range templateFiles {
		tmpl, err := newTemplate(templateFile, format)
		if err != nil {
			return "", err
		}
		out, err := tmpl.Render(config)
		if err != nil {
			return "", err
		}
		// normalizing output between template engines
		var buf bytes.Buffer
		if err := yamlext.UnmarshalSlice(out, func(in []byte) error {
			m := make(map[string]interface{})
			if err = yaml.Unmarshal(in, &m); err != nil {
				return err
			}
			o, err := yaml.Marshal(m)
			if err != nil {
				return err
			}
			buf.Write([]byte("---\n"))
			buf.Write(o)
			return nil
		}); err != nil {
			return "", err
		}
		output = append(output, buf.String())
	}
	return strings.Join(output, "---\n"), nil
}

func newTemplate(file string, format string) (tpl.Template, error) {
	content, err := readFile(file)
	if err != nil {
		return nil, err
	}
	if hasExtensionAny(file, ".yml.kubetpl", ".yaml.kubetpl", ".json.kubetpl-go") ||
		format == "placeholder" {
		return tpl.NewPlaceholderTemplate(content)
	}
	if hasExtensionAny(file, ".yml.kubetpl-go", ".yaml.kubetpl-go", ".json.kubetpl-go") ||
		format == "go-template" {
		return tpl.NewGoTemplate(content, file)
	}
	if hasExtensionAny(file, ".yml", ".yaml", ".json") ||
		format == "template-kind" {
		return tpl.NewTemplateKindTemplate(content)
	}
	if format != "" {
		return nil, fmt.Errorf("Unknown template type \"%s\"", format)
	}
	return nil, fmt.Errorf("Unable to infer type of \"%s\".\n"+
		"You either need to specify format explicitly with --type=<value> or "+
		"change the extension of the file to reflect the type.\n"+
		"See `kubetpl render --help` for more", file)
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
