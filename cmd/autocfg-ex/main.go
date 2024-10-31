package main

import (
	"fmt"
	"os"

	"github.com/davidwalter0/go-autocfg"
	"github.com/hashicorp/vault/api"
)

// App config options
type App struct {
	VaultAddr string `json:"vault-addr"`
	Role      string `json:"role"`
	Secret    string `json:"secret"`
	Filename  string `json:"filename,omitempty" doc:"filename for command line flag file name override"`
	Debug     bool   `json:"debug,omitempty"`
}

var app = &App{}
var text []byte

var client *api.Client

func main() {
	// var text []byte
	// var err error
	//  autocfg.Generator(app)
	autocfg.SetMode(autocfg.Union | autocfg.Indirect)
	fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.SearchModeName(autocfg.GetMode()))
	if app.Debug {
		fmt.Println(autocfg.String())
		fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.SearchModeName(autocfg.GetMode()))
	}
	autocfg.Configure(app)
	if app.Debug {
		autocfg.Dump(app)
	}

	app = &App{}
	autocfg.Reset()
	autocfg.SetMode(autocfg.Direct | autocfg.Union)
	fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.SearchModeName(autocfg.GetMode()))
	autocfg.Configure(app)
	// if text, err = json.MarshalIndent(app, "", "  "); err != nil {
	//  log.Fatal(err)
	// }
	if app.Debug {
		autocfg.Dump(app)
	}

	app = &App{}
	autocfg.Reset()
	autocfg.SetMode(autocfg.Indirect | autocfg.Union)
	fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.SearchModeName(autocfg.GetMode()))
	autocfg.Configure(app)
	// if text, err = json.MarshalIndent(app, "", "  "); err != nil {
	//  log.Fatal(err)
	// }
	if app.Debug {
		autocfg.Dump(app)
	}

	// fmt.Printf("%s\n", string(text))
}
