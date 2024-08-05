package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	models "github.com/nyudlts/go-medialog/models"
)

var (
	terms         = []string{}
	writer        bufio.Writer
	termFile      string
	root          string
	resourceCodes = []string{}
	oFile         string
)

func init() {
	flag.StringVar(&termFile, "term-file", "", "the location of the term-file")
	flag.StringVar(&root, "root", "", "the location of files to parse")
	flag.StringVar(&oFile, "output-file", "", "the location of output file")
}

func main() {

	flag.Parse()

	//parse the term file
	fmt.Println("* parsing term file")
	termFile, err := os.Open(termFile)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(termFile)
	for scanner.Scan() {
		terms = append(terms, scanner.Text())
	}
	fmt.Printf("* found %d terms", len(terms))

	fmt.Println("* creating output file")
	//create the output file
	of, _ := os.Create(oFile)
	defer of.Close()
	writer = *bufio.NewWriter(of)

	//create a hash for the header, write the header
	hdr := []string{"identifier", "total_count"}
	hdr = append(hdr, terms...)
	hdrString := strings.Join(hdr, "\t") + "\n"
	writer.Write([]byte(hdrString))
	writer.Flush()

	fmt.Println("* getting resources in medialog")
	//get resources in Medialog
	resp, err := http.Get("http://medialog.dlib.nyu.edu/api/v0/resources")
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	resources := []models.Resource{}
	if err := json.Unmarshal(body, &resources); err != nil {
		panic(err)
	}

	for _, resource := range resources {
		resourceCodes = append(resourceCodes, resource.CollectionCode)
	}

	fmt.Printf("* found %d resource in medialog\n", len(resourceCodes))

	//walk the directory

	fmt.Println("* parsing source files")
	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			searchFile(path)
		}
		return nil
	})

}

var prefixMap = map[string]string{
	"aia_":        "aia",
	"alba_":       "alba",
	"alba_video_": "albavideo",
	"films":       "films",
	"mss_":        "mss",
	"mc_":         "mc",
	"rg_":         "rg",
	"oh_":         "oh",
	"tam_":        "tam",
	"wag_":        "wag",
}

func searchFile(path string) {
	id := getID(path)

	if inMedialog(id) {
		fmt.Println("* in Medialog Skipping", id)
	} else {
		fmt.Printf("* parsing %s\n", id)
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Println("ERROR: ", err.Error())
		}

		line := []string{id}
		termMatches := map[string]int{}
		for _, term := range terms {
			matcher := regexp.MustCompile("(?i)" + term)
			m := matcher.FindAllIndex(b, -1)
			termMatches[term] = len(m)
		}

		var totalMatches = 0
		for _, v := range termMatches {
			totalMatches = totalMatches + v
		}

		if totalMatches > 0 {
			totalCount := strconv.Itoa(totalMatches)
			line = append(line, totalCount)
			for _, v := range termMatches {
				line = append(line, strconv.Itoa(v))
			}

			tsvLine := strings.Join(line, "\t")
			writer.WriteString(tsvLine + "\n")
			writer.Flush()
		}
	}

}

func getID(path string) string {
	id := strings.ReplaceAll(filepath.Base(path), ".xml", "")
	for k, v := range prefixMap {
		if strings.Contains(id, k) {
			id = strings.Replace(id, k, v, 1)
			if v == "rg" {
				id = strings.ReplaceAll(id, "_", "-")
			}
			return id
		}
	}

	return id
}

func inMedialog(s string) bool {
	for _, resource := range resourceCodes {
		if s == resource {
			return true
		}
	}
	return false
}
