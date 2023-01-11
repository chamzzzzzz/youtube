package main

import (
	"github.com/chamzzzzzz/youtube"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"time"
)

var (
	logger = log.New(os.Stdout, "youtube: ", log.Ldate|log.Lmicroseconds)
)

type App struct {
	cli      *cli.App
	client   *youtube.Client
	database *youtube.Database
	tz       string
	spec     string
	channels cli.StringSlice
}

func (app *App) Run() error {
	app.client = &youtube.Client{}
	app.database = &youtube.Database{}
	app.cli = &cli.App{
		Usage: "youtube collector and monitor",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "proxy",
				Usage:       "proxy",
				Destination: &app.client.Proxy,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_PROXY"},
			},
			&cli.StringSliceFlag{
				Name:        "channel",
				Usage:       "channel",
				Destination: &app.channels,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_CHANNEL"},
			},
			&cli.StringFlag{
				Name:        "dn",
				Value:       "mysql",
				Usage:       "database driver name",
				Destination: &app.database.DN,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_DN"},
			},
			&cli.StringFlag{
				Name:        "dsn",
				Value:       "root:root@/youtube",
				Usage:       "database source name",
				Destination: &app.database.DSN,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_DSN"},
			},
			&cli.StringFlag{
				Name:        "spec",
				Value:       "* 18 * * *",
				Usage:       "cron spec",
				Destination: &app.spec,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_SPEC"},
			},
			&cli.StringFlag{
				Name:        "tz",
				Value:       "Local",
				Usage:       "time zone",
				Destination: &app.tz,
				EnvVars:     []string{"YOUTUBE_COLLECTOR_TZ"},
			},
		},
		Action: app.run,
	}
	return app.cli.Run(os.Args)
}

func (app *App) run(c *cli.Context) error {
	if err := app.database.Migrate(); err != nil {
		return err
	}
	return app.cron()
}

func (app *App) cron() error {
	logger.Printf("monitoring.")
	c := cron.New(
		cron.WithLocation(location(app.tz)),
		cron.WithLogger(cron.VerbosePrintfLogger(logger)),
		cron.WithChain(cron.SkipIfStillRunning(cron.VerbosePrintfLogger(logger))),
	)
	c.AddFunc(app.spec, app.monitoring)
	c.Run()
	return nil
}

func (app *App) collect() ([]*youtube.Video, error) {
	var collected []*youtube.Video
	for _, channelID := range app.channels.Value() {
		videos, err := app.client.GetChannelVideos(channelID)
		if err != nil {
			return nil, err
		}
		for _, video := range videos {
			if has, err := app.database.HasVideo(video); err != nil {
				return nil, err
			} else if has {
				continue
			}

			if err := app.database.AddVideo(video); err != nil {
				return nil, err
			}
			collected = append(collected, video)
		}
	}
	return collected, nil
}

func (app *App) monitoring() {
	if videos, err := app.collect(); err != nil {
		logger.Printf("monitoring, err='%s'\n", err)
	} else {
		if len(videos) > 0 {
			logger.Printf("monitoring found new youtube video.")
			app.notification(videos)
		}
	}
}

func (app *App) notification(videos []*youtube.Video) {
	logger.Printf("send notification.")
}

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err != nil {
		return time.Local
	} else {
		return loc
	}
}

func main() {
	if err := (&App{}).Run(); err != nil {
		logger.Printf("run, err='%s'\n", err)
	}
}
