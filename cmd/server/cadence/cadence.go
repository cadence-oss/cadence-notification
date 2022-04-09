// Copyright (c) 2021 Cadence workflow OSS organization
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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	log.Printf("loaded config=\n%v\n", cfg.String())

	zapLogger, err := cfg.Log.NewZapLogger()
	if err != nil {
		log.Fatal("failed to create the zap logger, err: ", err.Error())
	}
	logger := loggerimpl.NewLogger(zapLogger)

	metricScope := cfg.Service.Metrics.NewScope(logger, "cadence-notification")

	svc, err := service.NewService(&cfg, logger, metricScope)
	if err != nil {
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

func getReceiverAddress(c *cli.Context) string {
	return strings.TrimSpace(c.GlobalString("receiver_address"))
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
		cli.StringFlag{
			Name: "receiver_address, ra",
			Value: ":8801",
			Usage: "receiver address",
			EnvVar: config.EnvKeyReceiverAddress,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "start",
			Aliases: []string{""},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "services",
					Value: "receiver, notifier",
					Usage: "start services/components in this project",
				},
			},
			Usage: "start cadence notification service",
			Action: func(c *cli.Context) {
				var wg sync.WaitGroup
				services := getServices(c)

				for _, service := range services {
					wg.Add(1)
					go launchService(service, c)
				}

				wg.Wait()
			},
		},
	}
	return app
}

func launchService(service string, c *cli.Context) {
	switch service {
	case "notifier":
		startHandler(c)
		break
	case "receiver":
		startTestWebhookEndpoint(c)
		break
	default:
		log.Printf("Invalid service: %v", service)
	}
}

func getServices(c *cli.Context) []string {
	val := strings.TrimSpace(c.String("services"))
	tokens := strings.Split(val, ",")

	if len(tokens) == 0 {
		log.Fatal("No services specified for starting")
	}

	services := []string{}
	for _, token := range tokens {
		t := strings.TrimSpace(token)
		services = append(services, t)
	}

	return services
}

func startTestWebhookEndpoint(c *cli.Context) {
	addr := getReceiverAddress(c)
	http.HandleFunc("/", logIncomingRequest)

	fmt.Printf("Starting server %s for testing...\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func logIncomingRequest(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		log.Printf("Path not supported: %v", r.URL.Path)
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "POST":
		var body []byte
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("[Failed to read request body]: %v", err.Error())
		}

		log.Printf("[Test server incoming request]: %v, URL: %v", string(body), r.URL.Path)
		w.WriteHeader(http.StatusOK)
		break
	default:
		log.Printf("Only POST methods are supported.")
		w.WriteHeader(http.StatusBadRequest)
	}
}
