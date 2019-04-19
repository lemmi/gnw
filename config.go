package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"reflect"
	"strings"
)

// Config holds user values and overrides
type Config struct {
	Hostname        string
	Description     string
	Lat             float64
	Lng             float64
	PositionComment string
	Contact         string
	Hood            string
	Distname        string
	Distversion     string
	Config          string
	Dry             bool
	Debug           bool
	Syslog          bool

	Log *log.Logger
}

func strOr(value, def string) string {
	if value == "" {
		value = def
	}
	return value
}
func floatOr(value, def float64) float64 {
	if value == 0 {
		value = def
	}
	return value
}
func configOr(conf, def Config) Config {
	conf.Hostname = strOr(conf.Hostname, def.Hostname)
	conf.Description = strOr(conf.Description, def.Description)
	conf.Lat = floatOr(conf.Lat, def.Lat)
	conf.Lng = floatOr(conf.Lng, def.Lng)
	conf.PositionComment = strOr(conf.PositionComment, def.PositionComment)
	conf.Contact = strOr(conf.Contact, def.Contact)
	conf.Hood = strOr(conf.Hood, def.Hood)
	conf.Distname = strOr(conf.Distname, def.Distname)
	conf.Distversion = strOr(conf.Distversion, def.Distversion)
	conf.Config = strOr(conf.Config, def.Config)
	conf.Syslog = conf.Syslog || def.Syslog

	return conf
}

func configFromCmd() Config {
	var c Config

	flag.StringVar(&c.Hostname, "hostname", "", "Hostname to report")
	flag.StringVar(&c.Description, "description", "", "Router description")
	flag.Float64Var(&c.Lat, "lat", 0, "Lateral position")
	flag.Float64Var(&c.Lng, "lng", 0, "Longitudinal position")
	flag.StringVar(&c.PositionComment, "positioncomment", "", "Position comment")
	flag.StringVar(&c.Contact, "contact", "", "Contact information")
	flag.StringVar(&c.Hood, "hood", "", "Name of the Hood")
	flag.StringVar(&c.Distname, "distname", "", "Name of the distribution")
	flag.StringVar(&c.Distversion, "distversion", "", "Version of the distribution")
	flag.StringVar(&c.Config, "config", "gateway.json", "Config file to load")
	flag.BoolVar(&c.Dry, "dry", false, "Don't send the report")
	flag.BoolVar(&c.Debug, "d", false, "Print debug information")
	flag.BoolVar(&c.Syslog, "syslog", false, "Use the syslog")

	flag.Parse()

	return c
}

func configFromFile(path string) (Config, error) {
	var c Config
	if path == "" {
		return c, nil
	}

	f, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
		return c, nil
	case err != nil:
		return c, err
	}
	defer f.Close()

	return c, json.NewDecoder(f).Decode(&c)
}

func configRequire(errors errors, option interface{}, name string) errors {
	zero := reflect.Zero(reflect.TypeOf(option))
	if option != zero.Interface() {
		return errors
	}
	err := fmt.Errorf("option %q is required", name)
	return append(errors, err)
}

type errors []error

func (es errors) Error() string {
	var ret strings.Builder

	for _, e := range es {
		fmt.Fprintln(&ret, e)
	}
	return ret.String()
}

func getConfig() (Config, error) {
	fromCmd := configFromCmd()
	fromFile, err := configFromFile(fromCmd.Config)
	if err != nil {
		return fromCmd, err
	}

	c := configOr(fromCmd, fromFile)
	hostname, _ := os.Hostname()
	c.Hostname = strOr(c.Hostname, hostname)

	var errors errors
	errors = configRequire(errors, c.Hostname, "Hostname")
	errors = configRequire(errors, c.Lat, "Lat")
	errors = configRequire(errors, c.Lng, "Lng")
	errors = configRequire(errors, c.Contact, "Contact")
	errors = configRequire(errors, c.Hood, "Hood")

	if c.Syslog {
		c.Log, err = syslog.NewLogger(syslog.LOG_NOTICE|syslog.LOG_DAEMON, log.LstdFlags)
		if err != nil {
			log.Println("Can't open syslog, falling back to normal logger")
			log.Println(err)
		}
	}
	if c.Log == nil {
		c.Log = log.New(os.Stdout, "gnw: ", log.LstdFlags)
	}

	if len(errors) == 0 {
		return c, nil
	}

	return c, errors
}
