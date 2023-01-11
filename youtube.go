package youtube

import (
	"database/sql"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	Proxy string
}

func (c *Client) GetFeed(feedURL string) (*Feed, error) {
	client := &http.Client{}
	if c.Proxy != "" {
		if proxyUrl, err := url.Parse(c.Proxy); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
		}
	}

	req, err := http.NewRequest("GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	body := &Feed{}
	if err := xml.Unmarshal(data, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) GetChannelVideos(channelID string) ([]*Video, error) {
	feed, err := c.GetFeed("https://www.youtube.com/feeds/videos.xml?channel_id=" + channelID)
	if err != nil {
		return nil, err
	}

	videos := make([]*Video, len(feed.Entry))
	for i, entry := range feed.Entry {
		video := &Video{
			ID:        strings.TrimPrefix(entry.ID, "yt:video:"),
			Title:     entry.Title,
			Published: entry.Published,
			ChannelID: channelID,
		}
		videos[i] = video
	}
	return videos, nil
}

type Database struct {
	DN  string
	DSN string
	db  *sql.DB
}

func (database *Database) getdb() (*sql.DB, error) {
	if database.db == nil {
		db, err := sql.Open(database.DN, database.DSN)
		if err != nil {
			return nil, err
		}
		database.db = db
	}
	return database.db, nil
}

func (database *Database) Close() {
	if database.db != nil {
		database.db.Close()
		database.db = nil
	}
}

func (database *Database) Migrate() error {
	db, err := database.getdb()
	if err != nil {
		return err
	}

	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS video (ID CHAR(32) NOT NULL PRIMARY KEY, ChannelID CHAR(32) NOT NULL, Title VARCHAR(256) NOT NULL, Published CHAR(32))"); err != nil {
		return err
	}
	return nil
}

func (database *Database) HasVideo(video *Video) (bool, error) {
	db, err := database.getdb()
	if err != nil {
		return false, err
	}

	rows, err := db.Query("SELECT ID FROM video WHERE ID = ?", video.ID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		return true, nil
	}
	return false, nil
}

func (database *Database) AddVideo(video *Video) error {
	db, err := database.getdb()
	if err != nil {
		return err
	}
	if _, err := db.Exec("INSERT INTO video(ID, ChannelID, Title, Published) VALUES(?,?,?,?)", video.ID, video.ChannelID, video.Title, video.Published); err != nil {
		return err
	}
	return nil
}

type Video struct {
	ID        string
	Title     string
	Published string
	ChannelID string
}

type Feed struct {
	Entry []*struct {
		ID        string `xml:"id"`
		Title     string `xml:"title"`
		Published string `xml:"published"`
	} `xml:"entry"`
}
