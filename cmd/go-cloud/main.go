package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/schoentoon/go-cloud/pkg/auth"
	gchttp "github.com/schoentoon/go-cloud/pkg/http"
	"github.com/schoentoon/go-cloud/pkg/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	var cfgfile = flag.String("config", "config.yml", "Config file location")
	flag.Parse()

	config, err := ReadConfig(*cfgfile)
	if err != nil {
		logrus.Fatal(err)
	}

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
	}

	logrus.Infof("Initializing database: %+v", config.DB)
	db, err := gorm.Open(sqlite.Open(config.DB), &gorm.Config{})
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Updating database models")
	err = models.InitModels(db)
	if err != nil {
		logrus.Fatal(err)
	}

	auth := auth.NewProvider(db)

	logrus.Info("Initializing plugin manager")
	pluginManager, err := config.Plugin.CreateManager()
	if err != nil {
		logrus.Fatal(err)
	}
	defer pluginManager.Close()

	logrus.Infof("Initializing storage provider %s", config.Storage.Provider)
	storage, err := config.Storage.CreateProvider(pluginManager)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Initializing file info providers")
	fileinfo, err := config.FileInfo.CreateProvider(pluginManager)
	if err != nil {
		logrus.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	logrus.Info("Initialing http server")
	server, err := gchttp.InitHttpServer(db, auth, storage, pluginManager, fileinfo)
	if err != nil {
		logrus.Fatal(err)
	}

	go func(server *http.Server, ch <-chan os.Signal) {
		<-c
		logrus.Info("Closing server")
		server.Close()
	}(server, c)

	err = server.ListenAndServe()
	if err != nil {
		logrus.Error(err)
	}
}
