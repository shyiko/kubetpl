package cli

import (
	"flag"
	"fmt"
	"github.com/posener/complete"
	"io"
	"os"
	"path/filepath"
)

type Completion struct{}

func (c *Completion) GenBashCompletion(w io.Writer) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "complete -C %s %s\n", bin, filepath.Base(bin))
	return nil
}

func (c *Completion) GenZshCompletion(w io.Writer) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "autoload +X compinit && compinit\nautoload +X bashcompinit && bashcompinit\ncomplete -C %s %s\n",
		bin, filepath.Base(bin))
	return nil
}

func (c *Completion) Execute() (bool, error) {
	bin, err := os.Executable()
	if err != nil {
		return false, err
	}
	run := complete.Command{
		Sub: complete.Commands{
			"completion": complete.Command{
				Sub: complete.Commands{
					"bash": complete.Command{},
					"zsh":  complete.Command{},
				},
			},
			"render": complete.Command{
				Flags: complete.Flags{
					"--allow-fs-access": complete.PredictNothing,
					"--chroot":          complete.PredictDirs("*"),
					"-c":                complete.PredictDirs("*"),
					"--freeze":          complete.PredictNothing,
					"-z":                complete.PredictNothing,
					"--freeze-list":     complete.PredictAnything,
					"--freeze-ref":      complete.PredictFiles("*"),
					"--input":           complete.PredictFiles("*"),
					"-i":                complete.PredictFiles("*"),
					"--output":          complete.PredictFiles("*"),
					"-o":                complete.PredictFiles("*"),
					"--set":             complete.PredictAnything,
					"-s":                complete.PredictAnything,
					"--syntax":          complete.PredictSet("$", "go-template", "kind-template"),
					"-x":                complete.PredictSet("$", "go-template", "kind-template"),
				},
				Args: complete.PredictFiles("*"),
			},
			"help": complete.Command{
				Sub: complete.Commands{
					"completion": complete.Command{
						Sub: complete.Commands{
							"bash": complete.Command{},
							"zsh":  complete.Command{},
						},
					},
					"render": complete.Command{},
				},
			},
		},
		Flags: complete.Flags{
			"--version": complete.PredictNothing,
		},
		GlobalFlags: complete.Flags{
			"--debug": complete.PredictNothing,
			"--help":  complete.PredictNothing,
			"-h":      complete.PredictNothing,
		},
	}
	run.Sub["r"] = run.Sub["render"]
	run.Sub["help"].Sub["r"] = run.Sub["help"].Sub["render"]
	completion := complete.New(filepath.Base(bin), run)
	if os.Getenv("COMP_LINE") != "" {
		flag.Parse()
		completion.Complete()
		return true, nil
	}
	return false, nil
}

func NewCompletion() *Completion {
	return &Completion{}
}
