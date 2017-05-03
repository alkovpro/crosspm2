package download

import (
	"os"
	"fmt"

	"github.com/alkovpro/crosspm2/config"
	"bufio"
	"io"
	"strings"
	"github.com/alkovpro/crosspm2/parse"
	"net/url"
)

type PackageList struct {
	urls []string
}

type CrossPMDownloader struct {
	cpmConfig     *config.CrossPMConfig
	cpmParser     parse.CrossPMParser
	ArtDownloader ArtifactoryDownloader
}

func NewDownloader(conf *config.CrossPMConfig) CrossPMDownloader {
	var dl CrossPMDownloader
	dl.cpmConfig = conf
	dl.cpmParser = parse.NewParser(dl.cpmConfig)
	dl.ArtDownloader = NewArtifactoryDownloader(dl.cpmConfig)
	return dl
}

func (dl *CrossPMDownloader) ReadDependencies(depsFile string) ([]map[string]string, error) {
	f, err := os.Open(depsFile)
	if err != nil {
		//fmt.Println(err)
		return []map[string]string{}, err
	}
	lines := make([]map[string]string, 0)
	var line string
	defer f.Close()
	r := bufio.NewReader(f)
	for i := 0; err != io.EOF; i++ {
		line, err = r.ReadString('\n')

		if len(line) > 0 {
			fields := strings.Fields(line)
			if (len(fields) > 0) && (fields[0][0] != '#') {
				cols := map[string]string{}
				for i, col := range fields {
					cols[dl.cpmConfig.Columns[i]] = col
				}
				lines = append(lines, cols)
			}
		}
	}
	if (err != io.EOF) && (err != nil) {
		//fmt.Println(err)
		return lines, err
	}
	return lines, nil
}

func hasItem(aStr *[]string, fStr string) bool {
	for _, item := range *aStr {
		if item == fStr {
			return true
		}
	}
	return false
}

func (dl *CrossPMDownloader) FindDependencies(depsFile string) {
	fmt.Println("Reading dependencies file:", depsFile)

	lines, err := dl.ReadDependencies(depsFile)

	if err != nil {
		fmt.Printf("Error reading dependencies file: %v\n", err)
		os.Exit(2)
	}

	pathParams := make(chan []parse.PathParam)
	jobs_prepare := 0

	for _, r := range lines {
		for _, src := range dl.cpmConfig.Config.Sources {
			for _, repo := range src.Repo {
				jobs_prepare++
				go func(parser string, values map[string]string, server string, repo string) { // , i int, ii int, iii int) {
					pathParams <- dl.cpmParser.FillPath(parser, values, server, repo)
					//fmt.Println(i, ii, iii)
				}(src.Parser, r, src.Server, repo) // , i, ii, iii)
			}
		}
	}
	fmt.Println("Search for files on server...")
	pathList := make(chan parse.PathsParam)
	jobs_search := 0
	for i := 0; i < jobs_prepare; i++ {
		repos := make([]string, 0)
		paths := make([]string, 0)
		names := make([]string, 0)
		params := make(map[string]string)
		for _, pp := range <-pathParams {
			params = pp.Params
			if !hasItem(&repos, pp.Params["repo"]) {
				repos = append(repos, pp.Params["repo"])
			}
			tmp, err := url.Parse(pp.Path)
			if err == nil {
				tmpPath := strings.Split(strings.Trim(tmp.Path, "/"), "/")
				if len(tmpPath) >= 3 {
					repo_path := ""
					if len(tmpPath) > 3 {
						// TODO: fix server removing
						repo_path = strings.Join(tmpPath[2:len(tmpPath)-1], "/")
						if !hasItem(&paths, repo_path) {
							paths = append(paths, repo_path)
						}
					}
					name := tmpPath[len(tmpPath)-1]
					if !hasItem(&names, name) {
						names = append(names, name)
					}
				}
			}
		}
		if (len(repos) > 0) && (len(paths) > 0) && (len(names) > 0) {
			jobs_search++
			go func(aRepo []string, aPath []string, aName []string) {
				pathList <- parse.PathsParam{
					Paths:  dl.ArtDownloader.AQLSearch(aRepo, aPath, aName),
					Params: params,
				}
			}(repos, paths, names)
		}

	}
	close(pathParams)

	for i := 0; i < jobs_search; i++ {
		item := <-pathList
		fmt.Println("\nParams(", i, "):")
		for item_name, item_value := range item.Params {
			fmt.Println("\t", item_name, "=",item_value)
		}
		fmt.Println("Paths(", i, "):")
		for _, item_path := range item.Paths {
			fmt.Println("\t", item_path)
		}
	}
	close(pathList)
	fmt.Println("Done!")
}
