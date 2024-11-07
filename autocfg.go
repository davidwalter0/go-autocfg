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
	"github.com/davidwalter0/go-tracer"
	"github.com/mitchellh/go-homedir"
)

var Trace = tracer.New()

func init() {
	Trace.Disable()
}

// LocalConfigFileName default for simple config lookup
var LocalConfigFileName = ".config.json"

// LocalAutoConfigFileName default for simple config lookup
var LocalAutoConfigFileName = ".autocfg.json"

// SetSimpleMode sets the mode and overrides the default
// @LocalConfigFileName
func SetSimpleMode(filename string) (err error) {
	_, err = SetMode(Simple)
	if err != nil {
		return
	}
	if len(filename) == 0 {
		err = fmt.Errorf("SetSimpleMode filename unset")
		return
	}
	if _, err = os.Stat(filename); errors.Is(err, fs.ErrNotExist) {
		err = fmt.Errorf("SetSimpleMode filename %s %w", filename, err)
		return
	}

	LocalConfigFileName = filename
	return
}

// Strict forces finding a configuration file
var Strict bool

/*
SearchMode selects the configuration search order and merge
strategy. Direct search ignores autocfg files. Indirect loads an
indirect path from which to load a configuration.

# Direct file paths

  - local: current working directory (.{{app name}}.json )
    when simple mode is set use .config.json
  - config: home .config directory (~/.config/{{app name}})/config.json
  - etc: /etc/ config directory (/etc/{{app name}})/config.json

# Auto config files

The indirect path in autocfg files are used to point to an alternate
configuration.

  - local: current working directory (.autocfg.json )
    when simple mode is set use .config.json
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
	// Simple mode uses .config.json in the current directory,
	// ignoring {{program name}}.json
	Simple
)

// var modes = []SearchMode{
//  Union | Direct,
//  Union | Indirect,
//  Union | Direct | Indirect,
//  First | Direct,
//  First | Indirect,
//  First | Direct | Indirect,
// }

// SearchModeMap maps a SearchMode to a text name of mode parts
var SearchModeMap = map[SearchMode]string{
	Direct:   "Direct",
	Indirect: "Indirect",
	// First is a short circuit evaluation i.e. stop evaluation after
	// success.
	First: "First",
	// Union merges each new config with the obj.
	Union:  "Union",
	Simple: "Simple",
}

// SearchModeName returns the string representation of the enum mask
func SearchModeName(mode SearchMode) (name string) {
	defer Trace.ScopedTrace()()
	if mode&Simple == Simple {
		return SearchModeMap[Simple]
	}
	for bit := 1; bit < 32; bit <<= 1 {
		//    fmt.Printf("%b %d %d %v\n", bit, bit, mode, int(mode)&bit)
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
var mode SearchMode = Simple //  | Indirect | Direct // Union | Direct

// SetMode and validate parameters for configure
func SetMode(m SearchMode) (sm SearchMode, err error) {
	defer Trace.ScopedTrace()()
	mode = SearchMode(m)
	if mode&First == First && mode&Union == Union {
		// Use the default mode
		mode = Union | Direct
		err = fmt.Errorf("First and Union Modes are mutually exclusive")
	}
	if mode&Simple == Simple {
		mode |= Direct
	}
	sm = mode
	return
}

// GetMode parameters for configure
func GetMode() (m SearchMode) {
	defer Trace.ScopedTrace()()
	m = mode
	return
}

// Mode alias for SearchModeName
func Mode() string {
	defer Trace.ScopedTrace()()
	return SearchModeName(mode)
}

// AutoCfg auto config format specifies the path of a configuration
// file to load and an environment variable map
type AutoCfg struct {
	Path string            `json:"path"   doc:"where to find the config spec file path"`
	Env  map[string]string `json:"env"    doc:"env var setup map[name]value"`
}

var pgm = strings.TrimSuffix(path.Base(os.Args[0]), path.Ext(os.Args[0]))

func generator(path string, obj any) (err error) {
	defer Trace.ScopedTrace()()
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
// example configuration. Replace prior definition when overwrite is
// true
func Generator(obj any, overwrite bool) {
	defer Trace.ScopedTrace()()
	var err error
	var path = "/tmp/dot.autocfg.json"
	var config = "/tmp/dot.config.json"

	// only

	if _, err = os.Stat(path); overwrite || errors.Is(err, fs.ErrNotExist) {
		generator(path, &AutoCfg{Path: config})
	}
	if _, err = os.Stat(config); overwrite || errors.Is(err, fs.ErrNotExist) {
		generator(config, obj)
	}
}

// LocalGenerator empty sample configuration files using the default
// autocfg type and an example object and place them in
// dot.autocfg.json pointing it's path to dot.config.json
// These can use used to confirm format and values of the arguments.
// Notice that if omitempty or similar json parameters are present in
// the tags the json Marshaling of the object are not included in the
// example configuration. Replace prior definition when overwrite is
// true
func LocalGenerator(obj any, overwrite bool) {
	defer Trace.ScopedTrace()()
	var err error
	var path = "dot.autocfg.json"
	var config = "dot.config.json"

	// only

	if _, err = os.Stat(path); overwrite || errors.Is(err, fs.ErrNotExist) {
		generator(path, &AutoCfg{Path: config})
	}
	if _, err = os.Stat(config); overwrite || errors.Is(err, fs.ErrNotExist) {
		generator(config, obj)
	}
}

func isPtr(obj any) (rc bool) {
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
	if !isPtr(obj) {
		return fmt.Errorf("arg obj any [%T]: object is not a pointer to struct", obj)
	}
	var found bool
	defer func() {
		defer Trace.ScopedTrace("!Strict")()
		if false {
			if !Strict {
				err = nil
			} else if !found {
				err = fmt.Errorf("%s %w", "no configuration found", err)
			}
		}
	}()
	var direct, indirect []string
	if Direct&mode == Direct {
		direct = DirectFiles()
		fmt.Println("direct", direct)
	}
	if Indirect&mode == Indirect {
		indirect = IndirectFiles()
		fmt.Println("indirect", indirect)
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
// First mode reverses the configuration file search order and stops
// on first configuration file found.
//
// - AUTOCFG_FILENAME when the env variable is set
// - .ex-app.json
// - ${HOME}/.config/ex-app/config.json
// - /etc/ex-app/config.json
//
// If AUTOCFG_FILENAME is set that file dominates and is processed
// in the order determined above.
//
// After the load of configuration from file(s) the env variables are
// processed. Any env variables replace configurations unmarshaled.
//
// After attributes are set from the environment, each corresponding
// flag argument is evaluated and replaces any value set in the struct
// with the command line argument from the flag the corresponding flag.
func Load(obj any, direct, indirect []string) (err error) {
	defer Trace.ScopedTrace()()
	// First mode is the same as short circuit evalutaion,
	var shortCircuit = mode&First == First
	if shortCircuit {
		for _, path := range direct {
			if err = LoadDirect(path, obj); err == nil {
				fmt.Fprintf(os.Stderr, "LoadDirect %v\n", err)
				if shortCircuit {
					return
				}
			}
		}
		for _, path := range indirect {
			if err = LoadIndirect(path, obj); err == nil {
				fmt.Fprintf(os.Stderr, "LoadIndirect %v\n", err)
				if shortCircuit {
					return
				}
			}
		}
	} else {
		for _, path := range indirect {
			_ = LoadIndirect(path, obj)
		}
		for _, path := range direct {
			_ = LoadDirect(path, obj)
		}
		// for _, path := range indirect {
		// 	if err = LoadIndirect(path, obj); err == nil {
		// 		fmt.Fprintf(os.Stderr, "LoadIndirect %v\n", err)
		// 	}
		// }
		// for _, path := range direct {
		// 	if err = LoadDirect(path, obj); err == nil {

		// 	}
		// }

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
	defer Trace.ScopedTrace()()
	var firstpaths = []string{}
	var unionpaths = []string{}
	var etc = fmt.Sprintf("/etc/%s/config.json", pgm)
	var config = fmt.Sprintf("${HOME}/.config/%s/config.json", pgm)
	var local = func() string {
		if mode&Simple == Simple {
			return LocalConfigFileName
		}
		return fmt.Sprintf(".%s.json", pgm)
	}()
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

var verbose bool
var debug bool = true

// Verbose logging for status info
func Verbose(v bool) {
	defer Trace.ScopedTrace()()
	verbose = v
}

// Debug verbose info
func Debug() bool {
	defer Trace.ScopedTrace()()
	return debug
}

// Dump an object via json MarshalIndent
func Dump(obj any) {
	defer Trace.ScopedTrace()()
	var err error
	var text []byte
	if text, err = json.MarshalIndent(obj, "", "  "); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", string(text))
}

// Configure an object automagically
func Configure(obj any) (err error) {
	defer Trace.ScopedTrace()()
	if err = configure(obj); err != nil {
		return
	}
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter configure\n")
		Dump(obj)
	}
	if err = cfg.Flags(obj); err != nil {
		log.Print(err)
		//    log.Fatal(err)
		if !Strict {
			err = nil
		}
		return
	}
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter cfg.Flags\n")
		Dump(obj)
	}
	return
}

// MultiCallConfigure an object automagically
func MultiCallConfigure(obj any) (err error) {
	defer Trace.ScopedTrace()()
	if err = configure(obj); err != nil {
		panic(err)
		return
	}
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter configure\n")
		Dump(obj)
	}
	cfg.Decorate()
	err = cfg.Nest(obj)
	cfg.Freeze()
	// if err = cfg.Nest(obj); err != nil {
	//  log.Print(err)
	//  //    log.Fatal(err)
	//  panic(err)
	//  if !Strict {
	//    err = nil
	//  }
	// }
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter cfg.Flags\n")
		Dump(obj)
	}
	return
}

// UnprefixedMultiCallConfigure an object automagically
func UnprefixedMultiCallConfigure(obj any) (err error) {
	defer Trace.ScopedTrace()()
	if err = configure(obj); err != nil {
		panic(err)
		return
	}
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter configure\n")
		Dump(obj)
	}
	//  cfg.Decorate()
	// var args = cfg.NewArg("")
	// args.Prefixed = true
	err = cfg.Unprefixed(obj)
	if err != nil {
		log.Print(err)
		//    log.Fatal(err)
	}
	//  err = cfg.Nest(obj)
	cfg.Freeze()
	// if err = cfg.Nest(obj); err != nil {
	//  log.Print(err)
	//  //    log.Fatal(err)
	//  panic(err)
	//  if !Strict {
	//    err = nil
	//  }
	// }
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter cfg.Flags\n")
		Dump(obj)
	}
	return
}

// PrefixMultiCallConfigure an object automagically with prefix to flag args
func PrefixMultiCallConfigure(prefix string, obj any) (err error) {
	defer Trace.ScopedTrace()()
	if err = configure(obj); err != nil {
		panic(err)
		return
	}
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter configure\n")
		Dump(obj)
	}
	if err = cfg.Reprefix(prefix, obj); err != nil {
		log.Fatal(err)
	}
	cfg.Freeze()
	//  panic(err)
	//  if !Strict {
	//    err = nil
	//  }
	// }
	if Debug() {
		fmt.Fprintf(os.Stderr, "\nafter cfg.Flags\n")
		Dump(obj)
	}
	return
}

// Usage from cfg tag parse with additional help text from the
// argument
func Usage(addText string) {
	defer Trace.ScopedTrace()()
	cfg.HelpText(addText)
	cfg.Usage()
}

// AutoConfigPath from the `autocfg.json` file in the {{application}}
// subdirectory of the users home .config directory
func AutoConfigPath() string {
	defer Trace.ScopedTrace()()
	homeDir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return path.Join(homeDir, ".config", pgm, "autocfg.json")
}

// LocalConfigPath from the `autocfg.json` file in the current work
// directory
func LocalConfigPath() string {
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
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
	for k, v := range autoCfg.Env {
		os.Setenv(k, v)
	}
	return
}

// LoadDirect read an application config file
func LoadDirect(path string, obj any) (err error) {
	defer Trace.ScopedTrace()()
	var text []byte
	if !isPtr(obj) {
		return fmt.Errorf("arg obj any [%T]: object is not a pointer to struct", obj)
	}
	path = ExpandEnvEvalTilde(path)

	if _, err = os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = nil
		}
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
	defer Trace.ScopedTrace()()
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// DirectFiles list of places to find a specified
// configuration
func DirectFiles() (paths []string) {
	defer Trace.ScopedTrace()()
	paths = []string{".config.json"}
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	var etc = fmt.Sprintf("/etc/%s/config.json", pgm)
	var config = fmt.Sprintf("${HOME}/.config/%s/config.json", pgm)
	var local = func() string {
		if mode&Simple == Simple {
			return LocalConfigFileName
		}
		return fmt.Sprintf(".%s.json", pgm)
	}()
	if len(ePath) > 0 {
		paths = append(paths, []string{ePath, local, config, etc}...)
	} else {
		paths = append(paths, []string{local, config, etc}...)
	}
	if mode&Union == Union {
		reverse(paths)
	}
	return
}

// IndirectFiles returns the list of auto config search paths
func IndirectFiles() (paths []string) {
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
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
	defer Trace.ScopedTrace()()
	text = fmt.Sprintf("Search mode = %s\n", SearchModeName(mode))

	text += fmt.Sprintf(`Direct load paths -- direct load paths are
configuration file names to attempt to load
`)
	if mode&Direct == Direct {
		for _, path := range DirectFiles() {
			text += fmt.Sprintf("\t%s\n", ExpandEnvEvalTilde(path))
		}
	}
	if mode&Indirect == Indirect {
		text += fmt.Sprintf(`Indirect load paths specifies files to load and
  extract the Path of a configuration file to load
`)
		for _, path := range IndirectFiles() {
			text += fmt.Sprintf("\t%s\n",
				ExpandEnvEvalTilde(path))
		}
	}
	return text
}

// autoCfgEnv string
func autoCfgEnv() (v string) {
	defer Trace.ScopedTrace()()
	for _, e := range os.Environ() {
		key, value, found := strings.Cut(e, "=")
		v += fmt.Sprintf("%s: %s %t\n", key, value, found)
	}
	return
}

// ExpandEnvEvalTilde expand ${var} and ~/
func ExpandEnvEvalTilde(path string) string {
	defer Trace.ScopedTrace()()
	var err error
	if path, err = homedir.Expand(os.ExpandEnv(path)); err != nil {
		fmt.Printf("homedir problem %v\n", err)
	}
	return path
}

// Reset flags for reconfigure
func Reset() {
	defer Trace.ScopedTrace()()
	cfg.Reset(pgm)
}
