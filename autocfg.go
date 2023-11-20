package autocfg

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

var pgm = strings.TrimSuffix(path.Base(os.Args[0]), path.Ext(os.Args[0]))

// AutoCfg auto config format specifies the path of a configuration
// file to load and an environment variable map
type AutoCfg struct {
	Path string            `json:"path"   doc:"where to find the config spec file path"`
	Env  map[string]string `json:"env"    doc:"env var setup map[name]value"`
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

// FindConfiguration checks for autoconfig spec file named in the env
// variable AUTOCFG_FILENAME, in the directory `.autocfg.json` or
// `~/.config/{{program}}/autocfg.json`
func FindConfiguration() (path string, err error) {
	var paths []string = AutoCfgFiles()
	for _, path = range paths {
		if _, err = os.Stat(path); err == nil {
			break
		}
	}
	return
}

// LoadConfiguration from an auto config path, returns nil on error
func LoadConfiguration(path string, obj any) (err error) {
	var text []byte
	if _, err = os.Stat(path); err == nil {
		text, err = os.ReadFile(path)
		if err == nil {
			text = []byte(os.ExpandEnv(string(text)))
			var autoCfg = &AutoCfg{}
			err = json.Unmarshal(text, autoCfg)
			if err == nil {
				if len(autoCfg.Path) > 0 {
					if _, err = os.Stat(autoCfg.Path); err == nil {
						text, err = os.ReadFile(autoCfg.Path)
						if err == nil {
							text = []byte(os.ExpandEnv(string(text)))
							err = json.Unmarshal(text, obj)
						}
					}
				} else {
					err = fmt.Errorf("%w empty config path", fs.ErrInvalid)
				}
			}
		}
	}
	return
}

// AutoCfgFiles returns the list of auto config search paths
func AutoCfgFiles() (paths []string) {
	paths = []string{}
	var ePath = os.Getenv("AUTOCFG_FILENAME")
	if len(ePath) > 0 {
		paths = []string{ePath, LocalConfigPath(), AutoConfigPath()}
	} else {
		paths = []string{LocalConfigPath(), AutoConfigPath()}
	}
	return paths
}

// AutoCfgFilesStringList
func ListAutoCfgFiles() (list string) {
	for _, path := range AutoCfgFiles() {
		list += fmt.Sprintf("%s\n", path)
	}
	return list
}

// autoCfgEnv string
func autoCfgEnv() (v string) {
	for _, e := range os.Environ() {
		key, value, found := strings.Cut(e, "=")
		v += fmt.Sprintf("%s: %s %t\n", key, value, found)
	}
	return
}
