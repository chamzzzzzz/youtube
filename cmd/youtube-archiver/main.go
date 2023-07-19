package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chamzzzzzz/youtube"
)

type Config struct {
	Channels    []string
	Destination string
}

type Stat struct {
	Channel  string
	Fetched  int
	Skipped  int
	Failed   int
	Archived int
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	if config.Destination == "" {
		config.Destination = "data"
	}

	err = os.MkdirAll(config.Destination, 0755)
	if err != nil {
		if !os.IsExist(err) {
			log.Fatal("Failed to create destination directory: ", err)
		}
	}

	client := &youtube.Client{}
	var stats []*Stat
	for _, channel := range config.Channels {
		dir := filepath.Join(config.Destination, channel)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			if !os.IsExist(err) {
				log.Printf("Failed to create channel [%s] directory [%s]. error:'%v'", channel, dir, err)
				continue
			}
		}

		videos, err := client.GetChannelVideos(channel)
		if err != nil {
			log.Printf("Failed to get videos for channel [%s]. error:'%v'", channel, err)
			continue
		}

		stat := &Stat{
			Channel: channel,
			Fetched: len(videos),
		}
		stats = append(stats, stat)

		groupByMonth := make(map[string][]*youtube.Video)
		for _, video := range videos {
			d, err := time.ParseInLocation(time.RFC3339, video.Published, time.Local)
			if err != nil {
				stat.Failed++
				log.Printf("Failed to parse video created_at [%s]. error:'%v'", video.Published, err)
				continue
			}
			date := d.Format("2006-01")
			groupByMonth[date] = append(groupByMonth[date], video)
		}

		for date, videos := range groupByMonth {
			file := filepath.Join(dir, date+".txt")
			_videos, err := load(file)
			if err != nil {
				stat.Failed += len(videos)
				log.Printf("Failed to load videos from file [%s]. error:'%v'", file, err)
				continue
			}

			skipped, archived := 0, 0
			for _, video := range videos {
				if has(_videos, video) {
					skipped++
					continue
				}
				_videos = append(_videos, video)
				archived++
			}
			if archived > 0 {
				err = save(file, _videos)
				if err != nil {
					stat.Failed += len(videos)
					log.Printf("Failed to save videos to file [%s]. error:'%v'", file, err)
					continue
				}
			}
			stat.Skipped += skipped
			stat.Archived += archived
		}

	}
	for _, stat := range stats {
		log.Printf("Channel [%s] stats: fetched:%d archived:%d skipped:%d failed:%d", stat.Channel, stat.Fetched, stat.Archived, stat.Skipped, stat.Failed)
	}
}

func loadConfig() (*Config, error) {
	b, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = json.Unmarshal(b, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func load(file string) ([]*youtube.Video, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var videos []*youtube.Video
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		video := &youtube.Video{}
		fields := strings.SplitN(line, "|", 3)
		if len(fields) != 3 {
			continue
		}
		video.Published = fields[0]
		video.ID = fields[1]
		video.Title = fields[2]
		videos = append(videos, video)
	}
	return videos, nil
}

func save(file string, videos []*youtube.Video) error {
	sort.Slice(videos, func(i, j int) bool {
		di, ei := time.ParseInLocation(time.RFC3339, videos[i].Published, time.Local)
		dj, ej := time.ParseInLocation(time.RFC3339, videos[j].Published, time.Local)
		if ei != nil || ej != nil {
			return videos[i].ID < videos[j].ID
		}
		return di.Before(dj)
	})
	var lines []string
	for _, video := range videos {
		lines = append(lines, strings.Join([]string{video.Published, video.ID, strip(video.Title)}, "|"))
	}
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0644)
}

func has(videos []*youtube.Video, video *youtube.Video) bool {
	for _, _video := range videos {
		if strip(_video.Title) == strip(video.Title) {
			return true
		}
	}
	return false
}

func strip(text string) string {
	return strings.ReplaceAll(text, "\n", "\\n")
}
