package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type LogTemplate struct {
	Regex         string `yaml:"regex"`
	Uuid          string `yaml:"uuid"`
	compiledRegex *regexp.Regexp
	Name          string `yaml:"name"`
	Severity      int    `yaml:"severity"`
}

type LogAnalyzer struct {
	Comment   string        `yaml:"comment"`
	File      string        `yaml:"file"`
	Templates []LogTemplate `yaml:"templates"`
}

// to be replacer with proper prometheur counters
var counters map[string]int = map[string]int{}

func readConfig(filename string) (*LogAnalyzer, error) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	result := &LogAnalyzer{}
	err = yaml.Unmarshal(yamlFile, result)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %w", filename, err)
	}
	for i := 0; i < len(result.Templates); i++ {
		if result.Templates[i].Uuid == "" {
			return nil, fmt.Errorf("empty uuid for template %q: ", result.Templates[i].Name)
		}
		if _, ok := counters[result.Templates[i].Uuid]; ok {
			return nil, fmt.Errorf("identical uuid for two regex: %q", result.Templates[i].Uuid)
		}
		counters[result.Templates[i].Uuid] = 0
		result.Templates[i].compiledRegex, err = regexp.Compile(result.Templates[i].Regex)
		if err != nil {
			return nil, fmt.Errorf("unable to compile regex %q in file %q", result.Templates[i].Regex, filename)
		}
	}
	return result, err
}

func analyzeLog(line string, config *LogAnalyzer) {
	byteLine := []byte(line)
	for i := 0; i < len(config.Templates); i++ {
		r := config.Templates[i].compiledRegex.Find(byteLine)
		if len(r) > 0 {
			counters[config.Templates[i].Uuid] += 1
			return
		}
	}
	fmt.Println(line)
	counters["<unmatched>"] += 1
}

func parseLogs(filename string, config *LogAnalyzer) error {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return fmt.Errorf("unable to open log file %q: %w", filename, err)
	}
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		analyzeLog(line[:len(line)-1], config)
	}
	return nil
}

func printResult() {
	for key, value := range counters {
		log.Printf("Key:%q => Element:%d\n", key, value)
	}
}

func main() {
	config, err := readConfig("templates/dpkg.yaml")
	if err != nil {
		log.Fatal(err)
	}
	err = parseLogs("/var/log/dpkg.log", config)
	if err != nil {
		log.Fatal(err)
	}
	printResult()
}
