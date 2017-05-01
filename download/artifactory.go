package download

import (
	"os"
	"fmt"
	"strings"
	"bytes"
	"encoding/json"

	"github.com/olekukonko/tablewriter"
	artifactory "github.com/lusis/go-artifactory/src/artifactory.v491"

	"github.com/alkovpro/crosspm2/config"
)

type ArtifactoryDownloader struct {
	cpmConfig *config.CrossPMConfig
	artConfig artifactory.ClientConfig
	artClient artifactory.ArtifactoryClient
}

func NewArtifactoryDownloader(conf *config.CrossPMConfig) ArtifactoryDownloader {
	var dl ArtifactoryDownloader
	dl.cpmConfig = conf
	dl.checkArtClient()
	return dl
}

func (dl *ArtifactoryDownloader) checkArtClient() {
	if dl.artConfig.BaseURL == "" {
		dl.artConfig = artifactory.ClientConfig{
			BaseURL:    dl.cpmConfig.Config.Common.Server,
			Username:   dl.cpmConfig.Config.Common.Auth[0],
			Password:   dl.cpmConfig.Config.Common.Auth[1],
			Token:      "",
			AuthMethod: "basic",
			VerifySSL:  false,
			Client:     nil,
			Transport:  nil,
		}
		dl.artClient = artifactory.NewClient(&dl.artConfig)
	}
}

func (dl *ArtifactoryDownloader) ListRepos(kind string) {
	fmt.Println("Fetch repos from server...")
	dl.checkArtClient()

	data, err := dl.artClient.GetRepos(kind)
	if err != nil {
		fmt.Println(err)
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{
			"Key",
			"Type",
			"Description",
			"Url",
		})
		for _, r := range data {
			table.Append([]string{
				r.Key,
				r.Rtype,
				r.Description,
				r.Url,
			})
		}
		table.Render()
	}
}

func (dl *ArtifactoryDownloader) ListFiles(repo string) {
	fmt.Println("Fetch files list from server...")
	dl.checkArtClient()

	data, err := dl.artClient.ListFiles(repo)
	if err != nil {
		fmt.Println(err)
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{
			"URI",
			"Size",
			"SHA-1",
		})
		for _, v := range data.Files {
			table.Append([]string{v.URI, fmt.Sprintf("%d", v.Size), v.SHA1})
		}
		table.Render()
	}
}

func (dl *ArtifactoryDownloader) GAVCSearch(repos []string, artifactID string) {
	fmt.Println("Search for files on server...")
	dl.checkArtClient()

	var gavc artifactory.Gavc
	gavc.ArtifactID = artifactID
	gavc.Repos = repos
	data, err := dl.artClient.GAVCSearch(&gavc)
	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.SetBorder(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		for _, r := range data {
			var innerBuf bytes.Buffer
			innerTable := tablewriter.NewWriter(&innerBuf)
			innerTable.SetHeader([]string{
				"File",
				"Repo",
				"RemoteUrl",
				"Created",
				"Last Modified",
				"Created By",
				"Modified By",
				"SHA1",
				"MD5",
				"Size",
				"MimeType",
			})
			elems := strings.Split(r.Path, "/")
			fileName := elems[len(elems)-1]
			innerTable.Append([]string{
				fileName,
				r.Repo,
				r.RemoteUrl,
				r.Created,
				r.LastModified,
				r.CreatedBy,
				r.ModifiedBy,
				r.Checksums.SHA1,
				r.Checksums.MD5,
				r.Size,
				r.MimeType,
			})
			innerTable.Render()
			table.Append([]string{
				innerBuf.String(),
			})
			table.Append([]string{
				fmt.Sprintf("Download: %s\n", r.Uri),
			})

		}
		table.Render()
	}
}

func (dl *ArtifactoryDownloader) AQLSearch(aRepo []string, aPath []string, aName []string) []string {
	//fmt.Println(aRepo)
	//fmt.Println(aPath)
	//fmt.Println(aName)

	var request artifactory.ArtifactoryRequest
	request.Verb = "POST"
	request.Path = "/api/search/aql"
	repos := ``
	paths := ``
	names := ``
	if len(aRepo) > 0 {
		for _, item := range aRepo {
			repos += fmt.Sprintf(`
				  {"repo":{"$eq":"%s"}},`,
				item)
		}
	} else {
		for _, src := range dl.cpmConfig.Config.Sources {
			for _, item := range src.Repo {
				repos += fmt.Sprintf(`
				  {"repo":{"$eq":"%s"}},`,
					item)
			}
		}
		for _, repo := range dl.cpmConfig.Config.Common.Repo {
			repos += fmt.Sprintf(`
			  {"repo":{"$eq":"%s"}},`,
				repo)
		}
	}
	if len(repos) > 0 {
		repos = repos[:len(repos)-1]
	}
	if len(aPath) > 0 {
		for _, item := range aPath {
			paths += fmt.Sprintf(`
				  {"path":{"$match":"%s"}},`,
				item)
		}
		paths = paths[:len(paths)-1]
		paths = fmt.Sprintf(`,
			"$or": [%s
			]`, paths)
	}
	if len(aName) > 0 {
		for _, item := range aName {
			names += fmt.Sprintf(`
				  {"name":{"$match":"%s"}},`,
				item)
		}
		names = names[:len(names)-1]
	}
	aqlString := fmt.Sprintf(`
		items.find(
		  {
		    "$or": [%s
			]%s,
			"$or": [%s
			]
		  }
		).include(
			"updated","created_by","repo","type","actual_md5","property.key","size","original_sha1","name",
			"modified_by","original_md5","property.value","path","modified","id","actual_sha1","created","depth"
		)`,
		repos,
		paths,
		names,
	)
	request.Body = bytes.NewReader([]byte(aqlString))
	request.ContentType = "text/plain"

	result := make([]string, 0)
	data, err := dl.artClient.HttpRequest(request)
	if err != nil {
		fmt.Println(err)
		return result
	}
	var dat artifactory.AQLResults
	err = json.Unmarshal(data, &dat)
	if err != nil {
		fmt.Println(err)
		return result
	}

	for _,v := range dat.Results {
		repo_path := fmt.Sprintf("%s/%s/%s/%s",
			dl.cpmConfig.Config.Common.Server,
			v.Repo,
			v.Path,
			v.Name)
		result = append(result, repo_path)
	}
	return result
	//table := tablewriter.NewWriter(os.Stdout)
	//table.SetHeader([]string{
	//	"Repo",
	//	"Path",
	//	"Name",
	//	"Size",
	//	"Quality",
	//	"tcBuildId",
	//})
	//qP := "contract."
	//for _, v := range dat.Results {
	//	quality, buildId := "", ""
	//	for _, p := range v.Properties {
	//		switch {
	//		case p.Key == "tcBuildId":
	//			buildId = p.Value
	//		case (len(p.Key) > len(qP)) && (p.Key[:len(qP)] == qP):
	//			if len(quality) > 0 {
	//				quality += ", "
	//			}
	//			quality += fmt.Sprintf("%s=%s", p.Key[len(qP):], p.Value)
	//		}
	//	}
	//	table.Append([]string{v.Repo, v.Path, v.Name, fmt.Sprintf("%d", v.Size), quality, buildId})
	//}
	//table.Render()
}
