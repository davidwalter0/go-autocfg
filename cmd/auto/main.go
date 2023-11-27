package main

import (
	"fmt"
	"log"

	"github.com/davidwalter0/go-autocfg"

	"github.com/hashicorp/vault/api"
)

// App config options
type App struct {
	VaultAddr string `json:"vault-address"`
	Role      string `json:"role"`
	Secret    string `json:"secret"`
	Filename  string `json:"filename,omitempty" doc:"filename for command line flag file name override"`
	Debug     bool   `json:"debug,omitempty"`
}

var app = &App{}
var text []byte

var client *api.Client

func main() {
	autocfg.Configure(app)
	autocfg.Dump(app)
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
