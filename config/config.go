package config

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
	"strings"
	"unicode"
	"os"
)

type tmpSource struct {
	SrcType  string   `yaml:"type"`
	Server   string   `yaml:"server"`
	Repo     []string `yaml:"repo,omitempty"`
	AuthType string   `yaml:"auth_type"`
	Auth     []string `yaml:"auth"`
	Parser   string   `yaml:"parser"`
}

type tmpParser struct {
	Columns    map[string]string `yaml:"columns"`
	Sort       []string          `yaml:"sort"`
	Index      int               `yaml:"index"`
	Path       string            `yaml:"path"`
	Properties string            `yaml:"properties"`
}

type CrossPMConfigFile struct {
	Cpm struct {
		Dependencies     string `yaml:"dependencies"`
		DependenciesLock string `yaml:"dependencies-lock"`
	} `yaml:"cpm"`
	Values    map[string]map[string]string `yaml:"values"`
	Parsers   map[string]tmpParser         `yaml:"parsers"`
	ColumnStr string                       `yaml:"columns"`
	Defaults  map[string]string            `yaml:"defaults"`
	Common    tmpSource                    `yaml:"common"`
	Sources   []tmpSource                  `yaml:"sources"`
}

type CrossPMConfig struct {
	Config       CrossPMConfigFile
	Columns      []string
	NameColumn   string
	ConfigFile   string
	DepsLockFile string
}

func NewConfig(configFileName string, depsLockFileName string) (CrossPMConfig) {
	var conf CrossPMConfig
	conf.ConfigFile = configFileName
	conf.DepsLockFile = depsLockFileName
	fmt.Printf("Reading config: %s\n", configFileName)
	source, err := ioutil.ReadFile(configFileName)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		os.Exit(1)
	}
	err = yaml.Unmarshal([]byte(source), &conf.Config)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	for i, src := range conf.Config.Sources {
		if len(strings.TrimSpace(src.Server)) == 0 {
			conf.Config.Sources[i].Server = strings.TrimSpace(conf.Config.Common.Server)
		}
		if len(src.Repo) == 0 {
			conf.Config.Sources[i].Repo = conf.Config.Common.Repo
		}
		if len(src.Parser) == 0 {
			conf.Config.Sources[i].Parser = conf.Config.Common.Parser
		}
	}

	if len(conf.Config.ColumnStr) > 0 {
		conf.Columns = strings.FieldsFunc(conf.Config.ColumnStr,
			func(c rune) bool {
				return !unicode.IsLetter(c) && !unicode.IsNumber(c) && (c != '*')
			},
		)
		for i, col := range conf.Columns {
			if col[0] == '*' {
				conf.Columns[i] = col[1:]
				conf.NameColumn = col[1:]
			}
		}

	}
	return conf
}

func (obj *CrossPMConfig) ShowParsers() bool {
	for i := range obj.Config.Parsers {
		fmt.Print(i, " = ")
		fmt.Println(obj.Config.Parsers[i])
	}

	return true
}
