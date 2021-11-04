// Copyright (c) 2021 Cadence OSS
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cadence

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	cconfig "github.com/uber/cadence/common/config"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/urfave/cli"

	"github.com/cadence-oss/cadence-notification/common/config"
	"github.com/cadence-oss/cadence-notification/service"
)

// startHandler is the handler for the cli start command
func startHandler(c *cli.Context) {
	env := getEnvironment(c)
	zone := getZone(c)
	configDir := getConfigDir(c)

	log.Printf("Loading config; env=%v,zone=%v,configDir=%v\n", env, zone, configDir)

	var cfg config.Config
	err := cconfig.Load(env, configDir, zone, &cfg)
	if err != nil {
		log.Fatal("Config file corrupted.", err)
	}
	if cfg.Log.Level == "debug" {
		log.Printf("config=\n%v\n", cfg.String())
	}

	zapLogger, err := cfg.Log.NewZapLogger()
	if err != nil {
		log.Fatal("failed to create the zap logger, err: ", err.Error())
	}
	logger := loggerimpl.NewLogger(zapLogger)

	svc, err := service.NewService(&cfg, logger)
	if err != nil{
		log.Fatal("fail to create service", err)
	}
	svc.Start()
}

func getEnvironment(c *cli.Context) string {
	return strings.TrimSpace(c.GlobalString("env"))
}

func getZone(c *cli.Context) string {
	return strings.TrimSpace(c.GlobalString("zone"))
}

func getConfigDir(c *cli.Context) string {
	return constructPathIfNeed(getRootDir(c), c.GlobalString("config"))
}

func getRootDir(c *cli.Context) string {
	dirpath := c.GlobalString("root")
	if len(dirpath) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("os.Getwd() failed, err=%v", err)
		}
		return cwd
	}
	return dirpath
}

// constructPathIfNeed would append the dir as the root dir
// when the file wasn't absolute path.
func constructPathIfNeed(dir string, file string) string {
	if !filepath.IsAbs(file) {
		return dir + "/" + file
	}
	return file
}

// BuildCLI is the main entry point for the cadence server
func BuildCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "cadence notification service"
	app.Usage = "Cadence notification service"
	app.Version = "beta"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "root, r",
			Value:  ".",
			Usage:  "root directory of execution environment",
			EnvVar: cconfig.EnvKeyRoot,
		},
		cli.StringFlag{
			Name:   "config, c",
			Value:  "config",
			Usage:  "config dir is a path relative to root, or an absolute path",
			EnvVar: cconfig.EnvKeyConfigDir,
		},
		cli.StringFlag{
			Name:   "env, e",
			Value:  "development",
			Usage:  "runtime environment",
			EnvVar: cconfig.EnvKeyEnvironment,
		},
		cli.StringFlag{
			Name:   "zone, az",
			Value:  "",
			Usage:  "availability zone",
			EnvVar: cconfig.EnvKeyAvailabilityZone,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "start",
			Aliases: []string{""},
			Usage:   "start cadence notification service",
			Action: func(c *cli.Context) {
				startHandler(c)
			},
		},
	}

	return app

}
