package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var terms = []string{}
var writer bufio.Writer

func main() {

	termFile, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(termFile)
	for scanner.Scan() {
		terms = append(terms, scanner.Text())
	}

	of, _ := os.Create("term-matches.tsv")
	defer of.Close()
	writer = *bufio.NewWriter(of)
	hdr := []string{"identifier", "total_count"}
	hdr = append(hdr, terms...)
	hdrString := strings.Join(hdr, "\t") + "\n"
	writer.Write([]byte(hdrString))
	writer.Flush()

	filepath.Walk(os.Args[2], func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			searchFile(path)
		}
		return nil
	})

}

func searchFile(path string) {

	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
	}

	id := strings.ReplaceAll(filepath.Base(path), ".xml", "")
	id = strings.ReplaceAll(id, "mss_", "mss")
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
