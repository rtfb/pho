package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type config struct {
	DBConn       string `yaml:"db_conn"`
	StaticDir    string `yaml:"static_dir"`
	Favicon      string `yaml:"favicon"`
	Port         string `yaml:"port"`
	CookieSecret string `yaml:"cookie_secret"`
	Log          string `yaml:"log"`
	LogSQL       bool   `yaml:"log_sql"`
}

func readConfigs() *config {
	homeDir := ""
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error acquiring current user. That can't be good.")
		fmt.Printf("Err = %q", err.Error())
	} else {
		homeDir = usr.HomeDir
	}
	var conf config
	yml, err := ioutil.ReadFile(filepath.Join(homeDir, ".phorc"))
	if err != nil {
		fmt.Println(err.Error())
		return &conf
	}
	err = yaml.Unmarshal(yml, &conf)
	if err != nil {
		fmt.Println(err.Error())
	}
	return &conf
}
