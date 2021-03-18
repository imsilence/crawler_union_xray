package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func parseTargetFile(path string) []string {
	targets := []string{}

	fhandler, err := os.Open(path)
	if err != nil {
		log.Printf("[-]error parse target file: %s, error: %s", path, err)
	} else {
		defer fhandler.Close()
		reader := bufio.NewReader(fhandler)

		buffer := bytes.NewBuffer([]byte{})
		for {
			line, isPerfix, err := reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					log.Printf("[-]error read target file: %s, error: %s", path, err)
				}
				break
			}
			buffer.Write(line)
			if !isPerfix {
				targets = append(targets, buffer.String())
				buffer.Reset()
			}
		}
	}

	return targets
}

type Req struct {
	Url     string
	Method  string
	Headers map[string]string
	Data    string
	Source  string
}

func md5Text(txt string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(txt)))
}

func mkdir(path string) {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
}

func crawlergo(url, dir, crawlergoBin, chromeBin string) []Req {
	outputFile := fmt.Sprintf("%s/crawlergo/%s.json", dir, md5Text(url))
	mkdir(outputFile)
	var reqs []Req
	log.Printf("exec crawlergo: %s", url)
	cmd := exec.Command(crawlergoBin, "-c", chromeBin, "-f", "smart", "--output-mode", "json", "--output-json", outputFile, url)
	if _, err := cmd.CombinedOutput(); err == nil {
		fhandler, err := os.Open(outputFile)
		if err != nil {
			return reqs
		}
		defer fhandler.Close()

		var m struct {
			ReqList       []Req    `json:"req_list"`
			AllReqList    []Req    `json:"all_req_list"`
			AllDomainList []string `json:"all_domain_list"`
			SubDomainList []string `json:"sub_domain_list"`
		}

		decoder := json.NewDecoder(fhandler)
		if err := decoder.Decode(&m); err == nil {
			reqs = append(reqs, m.AllReqList...)
		} else {
			log.Printf("error parse json crawlergo output, error: %s", err)
		}
	} else {
		log.Printf("error exec crawlergo command, error: %s", err)
	}
	return reqs
}

func xray(url, dir, xrayBin string) {
	outputFile := fmt.Sprintf("%s/xray/%s.json", dir, md5Text(url))
	mkdir(outputFile)

	log.Printf("exec xray: %s", url)
	cmd := exec.Command(xrayBin, "webscan", "--basic-crawler", url, "--json-output", outputFile)
	if _, err := cmd.CombinedOutput(); err == nil {
	} else {
		log.Printf("error exec xray output, error: %s", err)
	}
}

func main() {
	var (
		target       string
		targetFile   string
		outputDir    string
		help, h      bool
		crawlergoBin string
		xrayBin      string
		chromeBin    string
	)

	flag.StringVar(&target, "target", "", "target")
	flag.StringVar(&targetFile, "target-file", "", "target file")
	flag.StringVar(&outputDir, "output-dir", "", "output dir")
	flag.StringVar(&crawlergoBin, "crawlergo-bin", "bin/crawlergo.exe", "crawlergo bin")
	flag.StringVar(&xrayBin, "xray-bin", "bin/xray.exe", "xray bin")
	flag.StringVar(&chromeBin, "chrome-bin", "bin/chromedriver", "chrome bin")
	flag.BoolVar(&help, "help", false, "help")
	flag.BoolVar(&h, "h", false, "help")

	flag.Usage = func() {
		fmt.Println(`Usage: webscan --target-file="" --target=""`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if h || help {
		flag.Usage()
		os.Exit(0)
	}

	targets := []string{}
	if targetFile != "" {
		targets = append(targets, parseTargetFile(targetFile)...)
	}

	if target != "" {
		targets = append(targets, target)
	}

	if outputDir == "" {
		outputDir = fmt.Sprintf("result/%s", time.Now().Format("2006-01-02_15-04-05"))
	}
	for _, target := range targets {
		for _, req := range crawlergo(target, outputDir, crawlergoBin, chromeBin) {
			xray(req.Url, outputDir, xrayBin)
		}
	}

	fmt.Println("output: ", outputDir)
}
