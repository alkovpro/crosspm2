package main

import (
	"fmt"

	"github.com/alkovpro/crosspm2/config"
	"github.com/alkovpro/crosspm2/download"
	//"github.com/olekukonko/tablewriter"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"os"
)

var (
	appName             = "CrossPM2 (Cross Package Manager) version: %s The MIT License (MIT)"
	app                 = kingpin.New("crosspm2", "Cross Package Manager")
	appConfig           = app.Flag("config", "Path to configuration file").Default("crosspm.yaml").String()
	cmdDownload         = app.Command("download", "Download packages")
	cmdDownloadDepsLock = cmdDownload.Flag("depslock-path", "Path to file with locked dependencies").Default("dependencies.txt.lock").String()
	cmdLock             = app.Command("lock", "Lock packages dependencies")
	cmdShow             = app.Command("show", "Show some information")
	cmdShowType         = cmdShow.Arg("type", "Type of information to show").Default("package").Enum("package")
	cmdShowName         = cmdShow.Arg("name", "Name to search").String()
	//kind        = cmdShow.Flag("kind", "Types of repos to show").Default("all").Enum("local", "remote", "virtual", "all")
	conf       config.CrossPMConfig
	downloader download.CrossPMDownloader
)

func initApp() {
	conf = config.NewConfig(*appConfig, *cmdDownloadDepsLock)
	downloader = download.NewDownloader(&conf)
}

func main() {
	appName = fmt.Sprintf(appName, config.Version)
	fmt.Println(appName)
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case cmdDownload.FullCommand():
		initApp()
		downloader.FindDependencies(conf.DepsLockFile)

	case cmdLock.FullCommand():
		initApp()
		//locker.FindDependencies("_test/001/dependencies.txt")

	case cmdShow.FullCommand():
		initApp()
		switch *cmdShowType {
		case "package":
			fmt.Println("Search for files on server...")
			downloader.ArtDownloader.AQLSearch([]string{""}, []string{""}, []string{*cmdShowName})
		}
	}

}
