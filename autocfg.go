package autocfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/davidwalter0/go-cfg"
	"github.com/mitchellh/go-homedir"
)

// Strict forces finding a configuration file
var Strict bool

/*
SearchMode selects the configuration search order and merge
strategy. Direct search ignores autocfg files. Indirect loads an
indirect path from which to load a configuration.

# Direct file paths

- local: current working directory (.{{app name}}.json )

- config: home .config directory (~/.config/{{app name}})/config.json

- etc: /etc/ config directory (/etc/{{app name}})/config.json

# Auto config files

The indirect path in autocfg files are used to point to an alternate
configuration.

- local: current working directory (.autocfg.json )

- config: home .config directory (~/.config/{{app name}})/autocfg.json

- etc: /etc/ config directory (/etc/{{app name}})/autocfg.json

When AUTOCFG_FILENAME is specified it alters the runtime
environment to ignore direct modes and will attempt to load from
the autocfg configuration file specified

All modes evaluate environment variables but replace their values when
any corresponding flag is set load a file or files.
*/
type Modes int

// SearchMode The mode selected
var SearchMode Modes = DirectUnionMode

func TestMap(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}
	personMap := map[string]Person{
		"Alice": {Name: "Alice", Age: 30},
		"Bob":   {Name: "Bob", Age: 25},
	}

	name, ok := personMap["Charlie"]

	if ok {
		fmt.Println("Found person with name:", name)
	} else {
		fmt.Println("Person with name 'Charlie' not found")
	}

}

// ModeName returns the string representation of the enum
func ModeName(mode Modes) (name string) {
	name = "Unknown or unset mode"
	name = modes[mode]
	// if name, ok := modes[mode]; ok {

	// }
	return
}

var modes = map[Modes]string{
	DirectAndIndirectMode:    "DirectAndIndirectMode",
	DirectFirstFoundMode:     "DirectFirstFoundMode",
	AutoConfigOnlyFirstFound: "AutoConfigOnlyFirstFound",
	EnvThenFlagsOnly:         "EnvThenFlagsOnly",
	DirectUnionMode:          "DirectUnionMode",
}

const (
	// DirectAndIndirectMode search both direct paths etc, config and local and indirect + AUTOCFG_FILENAME
	DirectAndIndirectMode = iota
	// DirectFirstFound searches local, config, etc paths and stops
	// on the first file found.
	DirectFirstFoundMode
	// AutoConfigOnlyFirstFound searches local, config, etc paths and
	// stops on the first file found.
	AutoConfigOnlyFirstFound
	// EnvThenFlagsOnly is a fallback to use the default configuration
	// cfg mode, evaluating environment variables and overriding any
	// environment settings when a corresponding flag is set.
	EnvThenFlagsOnly
	// DirectUnionMode search etc,config,local load each of them
	// with union dominant, last configuration attribute is dominant
	// replacing matches based on the attribute key.
	DirectUnionMode

/*
// Search order etc,config,local (not implemented)
DirectUnionMode
// Search order etc,config,local (not implemented)
DirectFirstFound
// Search order local (.{{app name}}.json) ~/.config/{{app
// name}}/config.json /etc/{{app name}}.json
DirectLocalConfigEtc
// Search order local ~/.config/{{app name}}/autocfg.json
AutoLocalConfigEtc
*/
)

// AutoCfg auto config format specifies the path of a configuration
// file to load and an environment variable map
type AutoCfg struct {
	Path string            `json:"path"   doc:"where to find the config spec file path"`
	Env  map[string]string `json:"env"    doc:"env var setup map[name]value"`
}

var pgm = strings.TrimSuffix(path.Base(os.Args[0]), path.Ext(os.Args[0]))

func generator(path string, obj any) (err error) {
	var file *os.File
	file, err = os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var text []byte
	text, err = json.MarshalIndent(obj, "", "  ")
	file.Write(text)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// Generator empty sample configuration files using the default
// autocfg type and an example object and place them in
// /tmp/dot.autocfg.json pointing it's path to /tmp/dot.config.json
// These can use used to confirm format and values of the arguments.
// Notice that if omitempty or similar json parameters are present in
// the tags the json Marshaling of the object are not included in the
// example configuration.
func Generator(obj any) {
	var err error
	var path = "/tmp/dot.autocfg.json"
	var config = "/tmp/dot.config.json"

	// only

	if _, err = os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		generator(path, &AutoCfg{Path: config})
	}
	if _, err = os.Stat(config); errors.Is(err, fs.ErrNotExist) {
		generator(config, obj)
	}
}

func isPtr(obj any) (rc bool) {
	var v reflect.Value = reflect.ValueOf(obj)
	// Get the type of the struct that the value points to.
	if v.Kind() == reflect.Ptr {
		var oType = v.Elem().Type()
		if oType.Kind() == reflect.Struct {
			rc = true
		}
	}
	return
}

// FoundPath during configure
var FoundPath string

// DirectAndIndirect searches and loads the first configuration file
// found
//
// - When set a file named in the environment variable AUTOCFG_FILENAME
// - .{{program-name}}.json in the current directory
// - ~/.config/{{program-name}}/config.json
// - /etc/{{program-name}}/config.json
func DirectAndIndirect(obj any) (found bool, err error) {
	for _, path := range DirectFiles() {
		path = ExpandEnvEvalTilde(path)
		fmt.Printf("LoadDirect %v\n", path)
		if err = LoadDirect(path, obj); err == nil {
			found = true
			fmt.Printf("Found LoadDirect %v\n", path)
			return
		}
	}
	for _, path := range IndirectFiles() {
		path = ExpandEnvEvalTilde(path)
		fmt.Printf("LoadIndirect %v\n", path)
		if err = LoadIndirect(path, obj); err == nil {
			found = true
			fmt.Printf("Found LoadIndirect %v\n", path)
			return
		}
	}
	return
}

// configure an object automagically
func configure(obj any) (err error) {
	var path string
	if !isPtr(obj) {
		return fmt.Errorf("arg obj any [%T]: object is not a pointer to struct", obj)
	}
	var found bool
	defer func() {
		if !Strict {
			err = nil
		} else if !found {
			err = fmt.Errorf("%s %w", "no configuration found", err)
		}
	}()
	switch SearchMode {
	case DirectAndIndirectMode:
		found, err = DirectAndIndirect(obj)
		return
	case DirectFirstFoundMode:
		return DirectFirstFound(obj)
	case DirectUnionMode:
		return DirectUnionDirect(obj)
	default:
		if path, err = FindConfiguration(); err != nil {
			if err = LoadDirect(path, obj); err == nil {
				fmt.Printf("Found LoadDirect %v\n", path)
				return
			}
			if err = LoadIndirect(path, obj); err == nil {
				fmt.Printf("Found LoadIndirect %v\n", path)
				return
			}

			// for _, path := range SearchCfgs {
			// 	if err = LoadDirect(path, obj); err == nil {
			// 		return
			// 	}
			// }
		}
		// if err = LoadIndirect(path, obj); err == nil {
		// 	return
		// 	//			err = fmt.Errorf("%s %w", path, err)
		// }
		// if !Strict {
		// 	err = nil
		// }
	}
	return
}

var debug bool

// Debug verbose info
func Debug() bool {
	return debug
}

// DirectUnionDirect searches 3 paths for each config file. Found
// files are unmarshaled to the object argument. Unmarshal obeys the
// rules of 'union dominance' replacing any attribute(s) set by the
// next unmarshaled configuration. The last attribute(s) unmarshaled
// dominate - replace prior unmarshaling calls.
//
// The application name is evaluated from the binary name AKA
// filepath.Base(os.Args[0])
//
// For an application named ex-app the files would be searched in the
// following order:
//
// - /etc/ex-app/config.json
// - ${HOME}/.config/ex-app/config.json
// - .ex-app.json
// - AUTOCFG_FILENAME when the env variable is set
//
// If AUTOCFG_FILENAME is set that file dominates and is processed
// last.
//
// After the load of configuration from file(s) the env variables are
// processed. Any env variables replace configurations unmarshaled.
//
// After attributes are set from the environment, each corresponding
// flag argument is evaluated and replaces the corresponding flag
// struct variable.
func DirectUnionDirect(obj any) (err error) {
	var paths = []string{}
	var etc = fmt.Sprintf("/etc/%s/config.json", pgm)
	var config = fmt.Sprintf("${HOME}/.config/%s/config.json", pgm)
	var local = fmt.Sprintf(".%s.json", pgm)
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	if Debug() {
		fmt.Println(etc)
		fmt.Println(config)
		fmt.Println(local)
		fmt.Println(ePath)
	}
	paths = []string{etc, config, local, ePath}
	for _, path := range paths {
		err = LoadDirect(path, obj)
	}
	return
}

// DirectFirstFound searches 3 paths for each config file. The first found
// file is unmarshaled to the object argument then returns.
//
// The application name is evaluated from the binary name AKA
// filepath.Base(os.Args[0])
//
// For an application named ex-app the search order of the files is,
// local first.
//
// If AUTOCFG_FILENAME is set that file is searched first.
//
// - AUTOCFG_FILENAME when the env variable is set
// - .ex-app.json
// - ${HOME}/.config/ex-app/config.json
// - /etc/ex-app/config.json
//
// After the load of configuration from file(s) the env variables are
// processed. Any env variables replace configurations unmarshaled.
//
// After attributes are set from the environment, each corresponding
// flag argument is evaluated and replaces the corresponding flag
// struct variable.
func DirectFirstFound(obj any) (err error) {
	var paths = []string{}
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	var etc = fmt.Sprintf("/etc/%s/config.json", pgm)
	var config = fmt.Sprintf("${HOME}/.config/%s/config.json", pgm)
	var local = fmt.Sprintf(".%s.json", pgm)
	if Debug() {
		fmt.Println(ePath)
		fmt.Println(local)
		fmt.Println(config)
		fmt.Println(etc)
	}
	paths = []string{ePath, local, config, etc}
	for _, path := range paths {
		if err = LoadDirect(path, obj); err == nil {
			return
		}
	}
	return
}

// Dump an object via json MarshalIndent
func Dump(obj any) {
	var err error
	var text []byte
	if text, err = json.MarshalIndent(obj, "", "  "); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(text))
}

// Configure an object automagically
func Configure(obj any) (err error) {
	if err = configure(obj); err != nil {
		panic(err)
		return
	}
	// fmt.Printf("\nafter configure\n")
	// Dump(obj)
	if err = cfg.Flags(obj); err != nil {
		log.Print(err)
		//		log.Fatal(err)
		panic(err)
		if !Strict {
			err = nil
		}

	}
	// fmt.Printf("\nafter cfg.Flags\n")
	// Dump(obj)
	return
}

// AutoConfigPath from the `autocfg.json` file in the {{application}}
// subdirectory of the users home .config directory
func AutoConfigPath() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return path.Join(homeDir, ".config", pgm, "autocfg.json")
}

// LocalConfigPath from the `autocfg.json` file in the current work
// directory
func LocalConfigPath() string {
	var path string
	var err error
	var cwd string
	if cwd, err = os.Getwd(); err != nil {
		panic(err)
	}
	path, err = filepath.Abs(filepath.Join(cwd, ".autocfg.json"))
	if err != nil {
		panic(err)
	}
	return path
}

// FindConfiguration checks for direct config files then autoconfig
// spec named in the env variable AUTOCFG_FILENAME, in the directory
// `.autocfg.json` or `~/.config/{{program}}/autocfg.json`
func FindConfiguration() (path string, err error) {
	var paths []string = SearchPaths()
	for _, path = range paths {
		path = ExpandEnvEvalTilde(path)
		if _, err = os.Stat(path); err == nil {
			break
		}
	}
	return
}

// LoadIndirect from an auto config path. Read an autocfg file,
func LoadIndirect(path string, obj any) (err error) {
	var text []byte
	if !isPtr(obj) {
		return fmt.Errorf("arg obj any [%T]: object is not a pointer to struct", obj)
	}
	if _, err = os.Stat(path); err != nil {
		return
	}

	if text, err = os.ReadFile(path); err != nil {
		return
	}
	text = []byte(os.ExpandEnv(string(text)))
	var autoCfg = &AutoCfg{}

	if err = json.Unmarshal(text, autoCfg); err != nil {
		return
	}
	if len(autoCfg.Path) == 0 {
		err = fmt.Errorf("%w empty config path", fs.ErrInvalid)
	}
	if autoCfg.Path, err = homedir.Expand(os.ExpandEnv(autoCfg.Path)); err != nil {
		return
	}
	if _, err = os.Stat(autoCfg.Path); err != nil {
		return
	}
	if text, err = os.ReadFile(autoCfg.Path); err == nil {
		text = []byte(os.ExpandEnv(string(text)))
		err = json.Unmarshal(text, obj)
	}
	return
}

// LoadDirect read an application config file
func LoadDirect(path string, obj any) (err error) {
	var text []byte
	if !isPtr(obj) {
		return fmt.Errorf("arg obj any [%T]: object is not a pointer to struct", obj)
	}
	path = ExpandEnvEvalTilde(path)

	if _, err = os.Stat(path); err != nil {
		return
	}
	if text, err = os.ReadFile(path); err == nil {
		text = []byte(os.ExpandEnv(string(text)))
		err = json.Unmarshal(text, obj)
		if Debug() {
			fmt.Printf("> LoadDirect %s %v\n", path, err)
		}
	}
	return
}

// DirectFiles list of places to find a specified
// configuration
func DirectFiles() (paths []string) {
	paths = []string{}
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	var etc = fmt.Sprintf("/etc/%s/config.json", pgm)
	var config = fmt.Sprintf("${HOME}/.config/%s/config.json", pgm)
	var local = fmt.Sprintf(".%s.json", pgm)
	if len(ePath) > 0 {
		paths = []string{ePath, local, config, etc}
	} else {
		paths = []string{local, config, etc}
	}
	return
}

// IndirectFiles returns the list of auto config search paths
func IndirectFiles() (paths []string) {
	paths = []string{}
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	if len(ePath) > 0 {
		paths = []string{ePath, AutoConfigPath(), LocalConfigPath()}
	} else {
		paths = []string{AutoConfigPath(), LocalConfigPath()}
	}
	return
}

// SearchPaths  as a string
func SearchPaths() (list []string) {
	list = []string{}
	for _, path := range DirectFiles() {
		list = append(list, ExpandEnvEvalTilde(path))
	}

	for _, path := range IndirectFiles() {
		list = append(list, ExpandEnvEvalTilde(path))
	}
	return
}

// String shows load order
func String() (text string) {

	text = fmt.Sprintf(`Direct load paths -- direct load paths are
configuration file names to attempt to load
`)

	for _, path := range DirectFiles() {
		text += fmt.Sprintf("\t%s\n", ExpandEnvEvalTilde(path))
	}

	text += fmt.Sprintf(`Indirect load paths specifies files to load and
	extract the Path of a configuration file to load
`)
	for _, path := range IndirectFiles() {
		text += fmt.Sprintf("\t%s\n",
			ExpandEnvEvalTilde(path))
	}
	return text
}

// autoCfgEnv string
func autoCfgEnv() (v string) {
	for _, e := range os.Environ() {
		key, value, found := strings.Cut(e, "=")
		v += fmt.Sprintf("%s: %s %t\n", key, value, found)
	}
	return
}

// ExpandEnvEvalTilde expand ${var} and ~/
func ExpandEnvEvalTilde(path string) string {
	var err error
	if path, err = homedir.Expand(os.ExpandEnv(path)); err != nil {
		fmt.Printf("homedir problem %v\n", err)
	}
	return path
}

// Reset flags for reconfigure
func Reset() {
	cfg.Reset(pgm)
}
