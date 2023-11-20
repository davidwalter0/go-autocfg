package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/davidwalter0/go-cfg"
	"github.com/hashicorp/vault/api"
)

// App config options
type App struct {
	// VaultAddr string // `env:"VAULT_ADDR" json:"address" yaml:"address"`
	VaultAddr string
	Role      string
	Secret    string
	Filename  string `doc:"configuration file with vault-address, role and secret values in json format\n\t"`
	Debug     bool
}

// ConfigFiles
// thinking that an alternative like the current directory
// .dir-locals.json might be used
func (app *App) ConfigFiles() []string {
	var err error
	var homedir string
	var pgm = path.Base(os.Args[0])
	var configFiles = []string{}
	if len(app.Filename) > 0 {
		configFiles = append(configFiles, app.Filename)
	}
	homedir, err = os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	var dir string
	dir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	var baseDirName = path.Base(dir)
	var appNameJson = pgm + ".json"
	configFiles = append(configFiles,
		path.Join(dir, "."+baseDirName+".json"),
		path.Join(dir, ".dir-local.json"),
		path.Join(dir, appNameJson),
		path.Join(homedir, appNameJson),
		path.Join(homedir, ".config", pgm, appNameJson),
	)
	return configFiles
}

func (app *App) defaultCfgFile() string {
	var configFiles = app.ConfigFiles()
	var filename string
	var err error
	for _, filename = range configFiles {
		if _, err = os.Stat(filename); err != nil {
			switch {
			case !errors.Is(err, fs.ErrNotExist):
				filename = ""
				continue
			}
		}
		break
	}
	var pgm = path.Base(os.Args[0])
	var basename = "." + pgm + ".json"
	cfg.HelpText(fmt.Sprintf(`%s
Login to vault with the configured options

When not configured with the command line or environment variables
auto-configuration searches for a json formatted file

{
    "vault-address": "https://vault...",
    "role": "ae6b...",
    "secret": "4f2d..."
}

order of configuration file names to search for auto-configuration

- argument --filename={{value}}
- env var FILENAME={{value}}
- {{current-directory}}/. + {{current-directory}} + .json e.g. .{{current-directory}}.json
- {{current-directory}}/.dir-local.json
- {{current-directory}}/.{{application-name}}.json
- ~/.{{application-name}}.json
- ~/.config/{{application-name}}/.{{application-name}}.json

%s

When a configuration file is found then auto configure by loading that
json file.

The --filename argument takes precedence.

For example if no FILENAME env variable or --filename flag is
specified and ~/%s is found it will be loaded.

Search paths:

%s

Otherwise use the flags or environment variables as specified in the help.

  `, pgm, basename, basename, strings.Join(configFiles, "\n")))
	return filename
}

var app = &App{}
var text []byte

func init() {
	var config = app.defaultCfgFile()
	var err = cfg.Flags(app)
	if err != nil {
		log.Fatal(err)
	}
	if len(app.VaultAddr) == 0 || len(app.Role) == 0 || len(app.Secret) == 0 {
		if len(config) == 0 {
			cfg.Usage()
			log.Fatal("Empty, missing, or invalid arguments.")
		}
		text, err = os.ReadFile(config)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(text, app)
		if err != nil {
			log.Fatal(err)
		}
	} else {

	}

	if app.Debug {
		text, err = json.MarshalIndent(app, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
	}
	// if _, err := os.Stat(); !errors.Is(err, fs.ErrNotExist) {
	// }

}

var client *api.Client

func main() {
	var err error
	var conf = api.DefaultConfig()
	client, err = api.NewClient(conf)
	client.SetAddress(app.VaultAddr)

	resp, err := client.Logical().Write("auth/approle/login", map[string]interface{}{
		"role_id":   app.Role,
		"secret_id": app.Secret,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Use resp.Auth.ClientToken for whatever you want
	if app.Debug {
		fmt.Printf("Vault token: %s\n", resp.Auth.ClientToken)
		fmt.Printf("\nauth: %+v\n", resp)
	}
	fmt.Printf("%s", resp.Auth.ClientToken)
}
