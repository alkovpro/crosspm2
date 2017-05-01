package parse

import (
	"github.com/alkovpro/crosspm2/config"
	"strings"
	"regexp"
)

type CrossPMParser struct {
	cpmConfig *config.CrossPMConfig
}

func NewParser(conf *config.CrossPMConfig) (CrossPMParser) {
	var pr CrossPMParser
	pr.cpmConfig = conf
	return pr
}

type PathParam struct {
	Path   string
	Params map[string]string
}

type PathParams struct {
	Params []PathParam
	Stubs  map[string][]string
}

func (pp *PathParams) addParam(key string, stub string, value string) {
	found := false
	valCmp := ""
	if len(pp.Params) > 0 {
		valCmp, found = pp.Params[0].Params[key]
	} else {
		pp.Params = make([]PathParam, 1)
		pp.Stubs = make(map[string][]string)
		pp.Params[0].Params = make(map[string]string)
	}

	if _, ok := pp.Stubs[key]; !ok {
		pp.Stubs[key] = make([]string, 0)
	}
	stubFound := false
	for _, v := range pp.Stubs[key] {
		if v == stub {
			stubFound = true
			break
		}
	}
	if !stubFound {
		pp.Stubs[key] = append(pp.Stubs[key], stub)
	}
	begin := 0
	end := len(pp.Params)
	if found {
		for _, tmp := range pp.Params {
			aa := tmp.Params[key]
			if aa == value {
				return
			}
		}

		for ; begin < end; begin++ {
			if pp.Params[begin].Params[key] == valCmp {
				tmp := PathParam{
					Path:   "",
					Params: make(map[string]string),
				}
				for k, v := range pp.Params[begin].Params {
					tmp.Params[k] = v
				}
				pp.Params = append(pp.Params, tmp)
			}
		}
	}

	for i := begin; i < len(pp.Params); i++ {
		pp.Params[i].Params[key] = value
	}

}

func (dl *CrossPMParser) FillPath(parser string, values map[string]string, server string, repo string) []PathParam {
	r, _ := regexp.Compile(`(\[.*?]|\{.*?})`)
	tmpPath := dl.cpmConfig.Config.Parsers[parser].Path

	cols := make(map[string][]string, 0)
	for _, col := range r.FindAllString(tmpPath, -1) {
		if len(col) > 1 {
			if ((col[0] == '{') && (col[len(col)-1] == '}')) || ((col[0] == '[') && (col[len(col)-1] == ']')) {
				colFields := strings.FieldsFunc(col[1:len(col)-1], func(c rune) bool { return c == '|' })
				cols[col] = colFields
			}
		}
	}

	params := PathParams{}
	for col, items := range cols {
		colName := col
		stub := col
		for i, item := range items {
			colVal := item
			if col[0] == '{' {
				colName = items[0]
				if i == 0 {
					if colName == "server" {
						colVal = server
					} else if colName == "repo" {
						colVal = repo
					} else {
						colVal = values[item]
					}
				}
			}
			params.addParam(colName, stub, colVal)
		}
	}

	for i := range params.Params {
		params.Params[i].Path = tmpPath
		for key, stubs := range params.Stubs {
			for _, stub := range stubs {
				replacer := strings.NewReplacer(stub, params.Params[i].Params[key])
				params.Params[i].Path = replacer.Replace(params.Params[i].Path)
			}
		}
	}

	return params.Params
}
