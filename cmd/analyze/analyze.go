package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/mcuadros/go-syslog.v2"
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
	Name      string        `yaml:"name"`
	Templates []LogTemplate `yaml:"templates"`
}

var counters map[string]prometheus.Counter = map[string]prometheus.Counter{}

// TODO check uuid valid name (no - for instance)
// TODO should we add name and/or uuid as part of label ???
func mustReadConfig(filename string) *LogAnalyzer {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	result := &LogAnalyzer{}
	err = yaml.Unmarshal(yamlFile, result)
	if err != nil {
		panic(fmt.Errorf("in file %q: %w", filename, err))
	}
	if result.Name == "" {
		panic(fmt.Errorf("name can't be empty"))
	}
	if len(result.Templates) == 0 {
		panic(fmt.Errorf("you must have at least one regex"))
	}
	for i := 0; i < len(result.Templates); i++ {
		if result.Templates[i].Uuid == "" {
			panic(fmt.Errorf("empty uuid for template %q: ", result.Templates[i].Name))
		}
		if _, ok := counters[result.Templates[i].Uuid]; ok {
			panic(fmt.Errorf("identical uuid for two regex: %q", result.Templates[i].Uuid))
		}
		result.Templates[i].compiledRegex, err = regexp.Compile(result.Templates[i].Regex)
		if err != nil {
			panic(fmt.Errorf("unable to compile regex %q in file %q", result.Templates[i].Regex, filename))
		}
	}
	return result
}

func analyzeLog(line string, config *LogAnalyzer) {
	byteLine := []byte(line)
	for i := 0; i < len(config.Templates); i++ {
		r := config.Templates[i].compiledRegex.Find(byteLine)
		if len(r) > 0 {
			counters[config.Templates[i].Uuid].Inc()
			return
		}
	}
	fmt.Println(line)
	counters["<unmatched>"].Inc()
}

func mustParseSyslog(channel syslog.LogPartsChannel, config *LogAnalyzer) {
	for line := range channel {
		analyzeLog(fmt.Sprintf("%v", line["content"]), config)
	}
}

func mustCreateProbes(config *LogAnalyzer) {
	for i := 0; i < len(config.Templates); i++ {
		counters[config.Templates[i].Uuid] = promauto.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_%s_total", config.Name, config.Templates[i].Uuid),
			Help: fmt.Sprintf("The total number of lines matching regex %s", config.Templates[i].Regex),
		})
	}
	counters["<unmatched>"] = promauto.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_unmatched_total", config.Name),
		Help: "The total number of unmatched log lines",
	})
}

func main() {
	portString := os.Getenv("LAZ_SYSLOGPORT")
	portNumber, err := strconv.Atoi(portString)
	if err != nil {
		portNumber = 0
	}
	syslogPort := flag.Int("syslogport", portNumber, "Set it different to 0 to listen as rsyslog")
	configFile := flag.String("config", os.Getenv("LAZ_CONFIGFILE"), "config file specifying the regex")
	port := flag.Int("port", 3054, "port to expose the metrics")
	flag.Parse()
	if *configFile == "" {
		panic("Specify the log file in LAZ_CONFIGFILE environment variable or with -config")
	}
	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigint
		os.Exit(1)
	}()
	config := mustReadConfig(*configFile)
	mustCreateProbes(config)
	if *syslogPort != 0 {
		channel := make(syslog.LogPartsChannel)
		handler := syslog.NewChannelHandler(channel)
		server := syslog.NewServer()
		server.SetFormat(syslog.RFC3164)
		server.SetHandler(handler)
		err := server.ListenUDP(fmt.Sprintf("0.0.0.0:%d", *syslogPort))
		if err != nil {
			panic(err)
		}
		err = server.Boot()
		if err != nil {
			panic(err)
		}
		go func() {
			mustParseSyslog(channel, config)
		}()
	}
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}
