package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"io/ioutil"
	"log"
	"order/lotuss"
	"order/work"
	"os"
	"strconv"
)

type ServerCfg struct {
	Url         string `yaml:"url"`
	Token       string `yaml:"token"`
	GenerateDir string `yaml:"generateDir"`
	ImportDir   string `yaml:"importDir"`
	ListenAddr  string `yaml:"listen_addr"`
}

func loadConfig(configFilePath string) *ServerCfg {
	serverCfg := ServerCfg{}

	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatal(err)
		return &serverCfg
	}
	err = yaml.Unmarshal(data, &serverCfg)
	if err != nil {
		log.Fatalf("error: %v", err)
		return &serverCfg

	}
	return &serverCfg
}
func main() {
	log.Println("start server...")
	app := cli.NewApp()
	app.Name = "tools"
	app.Usage = "a file-coin tools"

	app.Flags = []cli.Flag{
		// 有参数则用参数，没参数才会使用环境变量
		&cli.StringFlag{
			Name:  "m",
			Value: "0,1,2",
			Usage: "运行模式:0,1,2",
		},
		&cli.StringFlag{
			Name:  "p",
			Value: "路径",
			Usage: "path",
		},
	}
	app.Action = func(c *cli.Context) error {
		mod := c.String("m")
		modInt, err := strconv.ParseInt(mod, 10, 64)
		if err != nil {
			log.Fatalf("modint is failed:%v", err)
			return err
		}
		path := c.String("p")
		cfg := loadConfig(path)
		fmt.Println(cfg)

		err = lotuss.Setup(cfg.Url, cfg.Token)
		fmt.Println(modInt)
		if err != nil {
			log.Fatalf("Setup lotus is failed:%v", err)
			return err
		}
		if modInt == 0 { //path
			work.DoWorkCar(c.Context, cfg.ImportDir)
		}
		if modInt == 1 { //path
			work.DoWork(c.Context, cfg.GenerateDir, cfg.ImportDir)

		}

		return nil

	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
