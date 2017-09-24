package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/ini.v1"
	"github.com/garethr/kubeval/kubeval"
	log "github.com/Sirupsen/logrus"
	"bytes"
	"strings"
	"errors"
	"github.com/shyiko/kubetpl/tpl"
	"github.com/ghodss/yaml"
	"net/http"
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
		RunE:func(cmd *cobra.Command, args []string) error {
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
			out, err := render(args, config, format)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(out)
			return nil
		},
		Example: "  # render template in placeholder format\n" +
		"  kubetpl render svc-and-deploy.yml.ktpl -c staging.env -d KEY=VALUE\n" +
		"  kubetpl render svc-and-deploy.yml --format=placeholder -c staging.env -d KEY=VALUE\n" +
		"\n" +
		"  # render template in go-template format\n" +
		"  kubetpl render svc-and-deploy.yml.goktpl -c staging.yml -p KEY=VALUE\n" +
		"  kubetpl render svc-and-deploy.yml --format=go-template -c staging.yml -d KEY=VALUE",
	}
	renderCmd.Flags().StringVarP(&format, "format", "f", "",
		"Template format (\"placeholder\", \"go-template\")")
	renderCmd.Flags().StringArrayVarP(&configFiles, "config", "c", nil, "Config (data) file(s) (*.ya?ml, *.env)")
	renderCmd.Flags().StringArrayVarP(&configKeyValuePairs, "data", "d", []string{}, "<key>=<value> pair (takes precedence over data (config) file(s))")
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
		validationResult, err := kubeval.Validate(out, templateFile)
		if err != nil {
			return "", err
		}
		if len(validationResult) != 0 {
			// todo: wrap into ValidationError?
			msg := []string{fmt.Sprintf("%s:", templateFile)}
			for _, vr := range validationResult {
				for _, r := range vr.Errors {
					msg = append(msg, r.String())
				}
			}
			if len(msg) != 1 {
				return "", errors.New(strings.Join(msg, "\n"))
			}
		}
		output = append(output, string(out))
	}
	return strings.Join(output, "---\n"), nil
}

func newTemplate(templateFile string, format string) (tpl.Template, error) {
	content, err := readFile(templateFile)
	if err != nil {
		return nil, err
	}
	if hasSuffix(templateFile, ".yml.ktpl", ".yaml.ktpl") ||
		format == "placeholder" {
		return tpl.NewPlaceholderTemplate(content)
	}
	if hasSuffix(templateFile, ".yml.goktpl", ".yaml.goktpl") ||
		format == "go-template" {
		return tpl.NewGoTemplate(content)
	}
	if format != "" {
		return nil, fmt.Errorf("Unknown template format \"%s\"", format)
	}
	// todo: describe how to specify format explicitly
	return nil, fmt.Errorf("Unable to infer format of \"%s\"", templateFile)
}

func hasSuffix(str string, suffix... string) bool {
	for _, suffix := range suffix {
		if strings.HasSuffix(str, suffix) {
			return true
		}
	}
	return false
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
	if strings.HasSuffix(path, ".env") {
		return parseDotEnv(data)
	}
	return parseYAML(data)
}

func readFile(path string) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		res, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
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
