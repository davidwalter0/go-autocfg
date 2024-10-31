// -*- mode:go-ts;mode:go-playground -*-
// snippet of code @ 2023-11-22 11:35:17

// === Go Playground ===
// Execute the snippet with:                 Ctl-Return
// Provide custom arguments to compile with: Alt-Return
// Other useful commands:
// - remove the snippet completely with its dir and all files: (go-playground-rm)
// - upload the current buffer to playground.golang.org:       (go-playground-upload)

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidwalter0/go-autocfg"
	"github.com/hashicorp/vault/api"
	"github.com/mitchellh/go-homedir"
)

// var app = &App{}
var text []byte

var client *api.Client

func main() {
	var multicall = &Multicall{}
	// autocfg.Configure(call)
	autocfg.SetMode(autocfg.Union | autocfg.Indirect)
	if false {
		fmt.Println("autocfg.AutoConfigPath()", autocfg.AutoConfigPath())
		fmt.Println("autocfg.LocalConfigPath()", autocfg.LocalConfigPath())
		fmt.Println("autocfg.String()", autocfg.String())
		autocfg.Verbose(true)
	}
	autocfg.Generator(multicall, true)
	var x = "multicall"
	switch x {
	case "unprefix":
		autocfg.UnprefixedMultiCallConfigure(multicall)
	case "multicall":
		autocfg.MultiCallConfigure(multicall)
	case "prefix":
		autocfg.PrefixMultiCallConfigure(path.Base(os.Args[0]), multicall)
	}
	// multicall.Authn("token")
	// os.Exit(0)
	// autocfg.Usage(MulticallText())
	// if true {
	//  autocfg.Dump(multicall)
	// }
	// os.Exit(0)
	for _, cmd := range []string{ /*"github", "approle",*/ "token"} { //, "github"} {
		if len(multicall.Github.Token) == 0 && len(multicall.Github.TokenFile) > 0 {
			var text []byte
			var err error
			text, err = EvalFileRead(multicall.Github.TokenFile)
			if err != nil {
				Check(err)
			}
			if len(text) > 0 {
				multicall.Github.Token = string(text)
			}
		}
		// if len(multicall.Vault.Token) == 0 && len(multicall.Vault.TokenFile) > 0 {
		if len(multicall.Vault.TokenFile) > 0 {
			var text []byte
			var err error
			text, err = EvalFileRead(multicall.Vault.TokenFile)
			if err != nil {
				Check(err)
			}
			if len(text) > 0 {
				multicall.Vault.Token = string(text)
			}
		}
		// for _, cmd := range []string{"approle", "github", "vault"} {
		var secret, err = multicall.Authn(cmd)
		fmt.Printf("secret %+v %s\n", secret, err)
	}

	// autocfg.Configure(&call.Approle)
	// autocfg.Dump(call.Approle)

	// autocfg.Configure(app)
	// autocfg.Dump(app)
}

var Calls = []string{
	"approle",
	"github",
	"token",
	"help",
}

type Multicall struct {
	// Cmd                string `json:"cmd" doc:"multi-call command to run"`
	Debug              bool   `json:"debug,omitempty"`
	InsecureSkipVerify bool   `json:"insecure-skip-verify" default:"true"`
	VaultAddr          string `json:"vault-address"`
	Approle            `json:"approle" doc:"Approle authentication"`
	Github             `json:"github"  doc:"Github authentication"`
	Vault              Token  `json:"vault" doc:"Vault Token authentication"`
	Filename           string `json:"filename,omitempty" doc:"filename for command line flag file name override"`
	// AuthPath           string `json:"auth-path"`
}

// Approle config options
type Approle struct {
	Role   string `json:"role"`
	Secret string `json:"secret"`
	Mount  string `json:"mount" default:"approle" doc:"typically approle mount is similar to auth/approle/login or auth/approle_{org}/login`
}

// Github config options
type Github struct {
	Token     string `json:"token"`
	TokenFile string `json:"token-file" default:"${HOME}/.secrets/vault-ghe-token"`
	Mount     string `json:"mount" default:"github_viper-cog" doc:"typically github mount is similar to auth/github/login or auth/github_{org}/login `
}

// Token config options
type Token struct {
	Token     string `json:"token"`
	TokenFile string `json:"token-file" default:"${HOME}/.vault-token"`
}

// FileExists test for file
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Check(args ...interface{}) {
	if len(args) > 0 {
		err, ok := args[len(args)-1].(error)
		if ok && err != nil {
			fmt.Fprintln(os.Stderr, err)
			panic(err)
		}
	}
}

// Abs return the absolute file name after expanding ~ to home
func Abs(name string) string {
	t, err := homedir.Expand(os.ExpandEnv(name))
	if err != nil {
		Check(err)
	}
	t, err = filepath.Abs(t)
	if err != nil {
		Check(err)
	}
	return t
}

// EvalFileRead and expand env \$var \${var} from on text
func EvalFileRead(filename string) (text []byte, err error) {
	filename = strings.TrimSpace(Abs(os.ExpandEnv(filename)))
	if !FileExists(filename) {
		err = fmt.Errorf("file not found [%s]", filename)
		return
	}
	text, err = os.ReadFile(filename)
	if err != nil {
		return
	}
	text = []byte(os.ExpandEnv(string(text)))
	return
}

func MulticallText() (text string) {
	for _, call := range Calls {
		text += call + "\n"
	}
	return fmt.Sprintf(`
commands:

%s

Examples

Extract a token from kube secret in mentos format

env KUBECONFIG=~/.kube/config.osx kubectl --context=qa get -o json secret/vault-token | jq -r '.data."vault-token"|=@base64d|.data."vault-token"'|jq -r .clientToken| tr -d '\n' > ~/.vault-token

Extract a token from kube secret in mentos format and save to a local
file dot.data.clientToken

env KUBECONFIG=~/.kube/config.osx kubectl --context=qa get -o json secret/vault-token | jq -r '.data."vault-token"|=@base64d|.data."vault-token"'|jq -c | tee dot.data.clientToken|jq -r .clientToken| tr -d '\n' 

`, text)
}

func (app *Multicall) Authn(cmd string) (secret *api.Secret, err error) {
	// if len(os.Args) == 1 {
	//  autocfg.Usage(MulticallText())
	//  return
	// }
	var filename = strings.TrimSpace(Abs(os.ExpandEnv("~/.vault-token")))
	var conf = api.DefaultConfig()
	client, err = api.NewClient(conf)
	client.SetAddress(app.VaultAddr)
	switch cmd {
	case "approle":
		fmt.Fprintf(os.Stderr, "approle: called as %s %s", path.Base(os.Args[0]), cmd)
		var mount = fmt.Sprintf("auth/%s/login", app.Approle.Mount)
		secret, err = client.Logical().Write(mount, map[string]interface{}{
			"role_id":   app.Role,
			"secret_id": app.Secret,
		})
		if err != nil {
			log.Fatal(err)
		}
		// Use secret.Auth.ClientToken for whatever you want
		if app.Debug {
			fmt.Printf("Vault token: %s\n", secret.Auth.ClientToken)
			fmt.Printf("\nauth: %+v\n", secret)
		}
		fmt.Printf("\nToken\n%s\n", secret.Auth.ClientToken)
		client.SetToken(secret.Auth.ClientToken)
		err = os.WriteFile(filename, []byte(secret.Auth.ClientToken), 0600)
	case "github":
		fmt.Fprintf(os.Stderr, "github: called as %s %s", path.Base(os.Args[0]), cmd)
		var mount = fmt.Sprintf("auth/%s/login", app.Github.Mount)
		var options = map[string]interface{}{
			"token": app.Github.Token,
		}
		secret, err = client.Logical().Write(mount, options)
		if err != nil {
			return
		}
		fmt.Printf("\nToken\n%s\n", secret.Auth.ClientToken)
		client.SetToken(secret.Auth.ClientToken)
		err = os.WriteFile(filename, []byte(secret.Auth.ClientToken), 0600)
	case "token":
		fmt.Fprintf(os.Stderr, "token: called as %s %s", path.Base(os.Args[0]), cmd)
		client.SetToken(app.Vault.Token)
		secret, err = client.Auth().Token().LookupSelf()
		client.SetToken(app.Vault.Token)
	case "help":
		fmt.Fprintf(os.Stderr, "help: called as %s %s", path.Base(os.Args[0]), cmd)
		autocfg.Usage(MulticallText())
	default:
		autocfg.Usage(MulticallText())
	}
	if secret != nil {
		var text []byte
		text, err = json.MarshalIndent(secret.Data, "", "  ")
		fmt.Printf("\nToken\n%s\n", string(text))
		var timeLeft time.Duration
		timeLeft, err = secret.TokenTTL()
		if err != nil {
			err = fmt.Errorf("Vault: unable to lookup token details: %v", err)
			return
		}
		var isRenewable bool
		isRenewable, err = secret.TokenIsRenewable()
		if err != nil {
			err = fmt.Errorf("Vault: unable to lookup TokenIsRenewable: %v", err)
			return
		}
		if secret != nil && secret.Data != nil && secret.Data["period"] != nil {
			fmt.Fprintf(os.Stderr, "type %T %+v\n", secret.Data["period"], secret.Data["period"])
			var rw int64
			rw, err = secret.Data["period"].(json.Number).Int64()
			if err != nil {
				err = fmt.Errorf("Vault: unable to find period: %v", err)
			} else {
				refreshWindow = rw
			}
			fmt.Fprintf(os.Stderr, "type %T %+v %v\n", rw, rw, err)
		}

		if timeLeft.Seconds() < float64(refreshWindow) && !isRenewable {
			err = fmt.Errorf("Vault expired not renewable: %v", err)
			return
		}

		if timeLeft.Seconds() < float64(refreshWindow) {
			//      os.Setenv("VAULT_TOKEN", app.Vault.Token)
			// secret, err = client.Auth().Token().Renew(app.Vault.Token, refreshWindow)
			secret, err = client.Auth().Token().RenewSelf(0)
			// secret, err = client.Auth().Token().RenewSelf(refreshWindow)
			if err != nil {
				err = fmt.Errorf("Vault unable to renew vault token: %v", err)
				return
			}
		}
		secret, err = client.Auth().Token().LookupSelf()
		if secret != nil {
			text, err = json.MarshalIndent(secret.Data, "", "  ")
			fmt.Printf("\nToken\n%s\n", string(text))
		}
	}

	return
}

var (
	refreshWindow int64 = 86400
)
