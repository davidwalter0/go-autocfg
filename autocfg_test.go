package autocfg

import (
	"fmt"
	"os"
	"path"
	"testing"
)

type fakeTestConf struct {
	VaultAddr string `json:"vault-address"`
	Role      string
	Secret    string
	Filename  string
	Debug     bool
}

type testCfg struct {
	FakeHome                 string
	Dir                      string
	HomeAutoCfgFileName      string
	AutoCfgText              string
	FakeLocalAutoCfgJSONFile string
	CfgFilename              string
	CfgString                string
}

func TestAutoCfgVar(t *testing.T) {
	var localTestCfg = newTestCfg(t)
	var err error
	defer os.RemoveAll(localTestCfg.Dir)
	var foundPath string
	foundPath, err = FindConfiguration()
	if err != nil {
		t.Error(foundPath, err)
	}
	//	t.Log("[", foundPath, err, "]")
	var config = &fakeTestConf{}
	err = LoadConfiguration(foundPath, config)
	if err != nil {
		t.Error(foundPath, err)
	}
	//	t.Logf("[ %+v\n%v ]\n", config, err)
}

// func TestPrintList(t *testing.T) {
// 	t.Logf("\n%s\n", ListAutoCfgFiles())
// }

// func TestPrintEnv(t *testing.T) {
// 	t.Logf("\n%s\n", autoCfgEnv())
// }

func newTestCfg(t *testing.T) *testCfg {
	var dir, err = os.MkdirTemp("/tmp", "autocfg-test-*")
	var fakeHome = path.Join(dir, os.Getenv("HOME"))
	err = os.MkdirAll(fakeHome, 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(path.Join(fakeHome, ".config", "app"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	var testcfg = &testCfg{
		FakeHome:            fakeHome,
		Dir:                 dir,
		HomeAutoCfgFileName: fmt.Sprintf("%s/.autocfg.json", fakeHome),
		AutoCfgText: fmt.Sprintf(`
{
  "path": "%s/${HOME}/.config/app/config.json"
}
`, dir),
		FakeLocalAutoCfgJSONFile: fmt.Sprintf("%s/.autocfg.json", dir),
		CfgFilename:              path.Join(fakeHome, ".config/app/config.json"),
		CfgString: `
{
    "vault-address": "https://vault...",
    "role": "ae6b6697...",
    "secret": "4f2df009..."
}
`,
	}
	var file *os.File
	file, err = os.Create(testcfg.HomeAutoCfgFileName)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	file.WriteString(testcfg.AutoCfgText)
	file, err = os.Create(testcfg.CfgFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	file.WriteString(testcfg.CfgString)

	file, err = os.Create(testcfg.FakeLocalAutoCfgJSONFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	os.Setenv("AUTOCFG_FILENAME", testcfg.HomeAutoCfgFileName)
	file.WriteString(testcfg.AutoCfgText)

	return testcfg
}
