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
// type Modes int
type SearchMode int

const (
	// Union merges all found configuration files.
	Union SearchMode = 1 << iota
	// First is a short circuit evalutaion, stop on first found,
	First
	// Direct loads files from their names when found.
	Direct
	// Indirect loads an autocfg file, then uses it's path attribute to
	// find the configuration file to load.
	Indirect
)

// var modes = []SearchMode{
// 	Union | Direct,
// 	Union | Indirect,
// 	Union | Direct | Indirect,
// 	First | Direct,
// 	First | Indirect,
// 	First | Direct | Indirect,
// }

// SearchModeMap maps a SearchMode to a text name of mode parts
var SearchModeMap = map[SearchMode]string{
	Direct:   "Direct",
	Indirect: "Indirect",
	// First is a short circuit evaluation i.e. stop evaluation after
	// success.
	First: "First",
	// Union merges each new config with the obj.
	Union: "Union",
}

// SearchModeName returns the string representation of the enum mask
func SearchModeName(mode SearchMode) (name string) {
	for bit := 1; bit < 32; bit <<= 1 {
		fmt.Printf("%b %d %d %v\n", bit, bit, mode, int(mode)&bit)
		if int(mode)&bit > 0 {
			if len(name) > 0 {
				name += "-" + SearchModeMap[SearchMode(bit)]
			} else {
				name += SearchModeMap[SearchMode(bit)]
			}
		}
	}
	return
}

// mode The mode selected
var mode SearchMode = Union | Direct

// SetMode and validate parameters for configure
func SetMode(m SearchMode) (err error) {
	mode = SearchMode(m)
	if mode&First == First && mode&Union == Union {
		// Use the default mode
		mode = Union | Direct
		err = fmt.Errorf("First and Union Modes are mutually exclusive")
	}
	return
}

// GetMode and validate parameters for configure
func GetMode() (m SearchMode) {
	m = mode
	return
}

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
	// Is this a pointer to an object?
	if v.Kind() == reflect.Ptr {
		var oType = v.Elem().Type()
		// Does it point to a struct?
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
	var direct, indirect []string
	if Direct&mode == Direct {
		direct = DirectFiles()
	}
	if Indirect&mode == Indirect {
		indirect = IndirectFiles()
	}
	err = Load(obj, direct, indirect)
	return
}

// Load searches the paths provided
//
// When mode & (First | Direct ) return on the first configuration
// file found.
//
// When mode & (Union | Direct) then unmarshal each found
// configuration obeying rule of 'union dominance' replacing any
// attribute(s) set by the next unmarshaled configuration. The last
// attribute(s) unmarshaled dominate - replace prior unmarshaling
// calls.
//
// The application name is evaluated from the binary name AKA
// filepath.Base(os.Args[0])
//
// For an application named ex-app the files would be searched in the
// following order:
//
// Union mode:
//
// - /etc/ex-app/config.json
// - ${HOME}/.config/ex-app/config.json
// - .ex-app.json
// - AUTOCFG_FILENAME when the env variable is set
//
// First mode reverses the search order.
//
// - AUTOCFG_FILENAME when the env variable is set
// - .ex-app.json
// - ${HOME}/.config/ex-app/config.json
// - /etc/ex-app/config.json
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
func Load(obj any, direct, indirect []string) (err error) {
	// First mode is the same as short circuit evalutaion,
	var shortCircuit = mode&First == First
	for _, path := range direct {
		if err = LoadDirect(path, obj); err == nil {
			if shortCircuit {
				return
			}
		}
	}
	for _, path := range indirect {
		if err = LoadIndirect(path, obj); err == nil {
			if shortCircuit {
				return
			}
		}
	}
	return
}

// IndirectLoad searches 3 paths for an indirect autocfg config
// file. Found files are unmarshaled to an autocfg object argument.
// The file is then parsed for it's path argument pointing to a
// configuration file.
//
// When mode & (First | Indirect ) return on the first configuration
// file found.
//
// When mode & (Union | Indirect) then for each indirect config file
// found, unmarshal each found configuration obeying rule of 'union
// dominance' replacing any attribute(s) set by the next unmarshaled
// configuration. The last attribute(s) unmarshaled dominate - replace
// prior unmarshaling calls.
//
// The application name is evaluated from the binary name AKA
// filepath.Base(os.Args[0])
//
// For an application named ex-app the indirect files would be
// searched in the following order:
//
// Union mode:
//
// - /etc/ex-app/config.json
// - ${HOME}/.config/ex-app/config.json
// - .ex-app.json
// - AUTOCFG_FILENAME when the env variable is set
//
// First mode reverses the search order.
//
// - AUTOCFG_FILENAME when the env variable is set
// - .ex-app.json
// - ${HOME}/.config/ex-app/config.json
// - /etc/ex-app/config.json
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
func IndirectLoad(obj any) (err error) {
	var firstpaths = []string{}
	var unionpaths = []string{}
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
	unionpaths = []string{etc, config, local, ePath}
	firstpaths = []string{ePath, local, config, etc}
	if mode&First == First {
		for _, path := range firstpaths {
			// First is the same as short circuit evalutaion,
			if err = LoadIndirect(path, obj); err == nil {
				return
			}
		}
	} else {
		for _, path := range unionpaths {
			err = LoadIndirect(path, obj)
		}
	}
	return
}

var debug bool

// Debug verbose info
func Debug() bool {
	return debug
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

// reverse slice
func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
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
	fmt.Fprintf(os.Stderr, "before if mode&First == First { \n\t%+v\n", paths)
	if mode&Union == Union {
		reverse(paths)
	}
	fmt.Fprintf(os.Stderr, "after if mode&First == First { \n\t%+v\n", paths)
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
	if mode&Union == Union {
		reverse(paths)
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
