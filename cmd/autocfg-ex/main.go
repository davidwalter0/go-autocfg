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
	//	autocfg.Generator(app)
	autocfg.SearchMode = autocfg.DirectUnionMode
	fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.ModeName(autocfg.SearchMode))
	if app.Debug {
		fmt.Println(autocfg.String())
		fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.ModeName(autocfg.SearchMode))
	}
	autocfg.Configure(app)
	if app.Debug {
		autocfg.Dump(app)
	}

	app = &App{}
	autocfg.Reset()
	autocfg.SearchMode = autocfg.DirectFirstFoundMode
	fmt.Fprintf(os.Stderr, "Mode %v\n", autocfg.ModeName(autocfg.SearchMode))
	autocfg.Configure(app)
	// if text, err = json.MarshalIndent(app, "", "  "); err != nil {
	// 	log.Fatal(err)
	// }
	if app.Debug {
		autocfg.Dump(app)
	}
	// fmt.Printf("%s\n", string(text))
}
