package youtube

import (
	"os"
	"testing"
)

func TestGetChannelVideos(t *testing.T) {
	var c = Client{
		Proxy: os.Getenv("YOUTUBE_CLIENT_TEST_PROXY"),
	}
	channelID := os.Getenv("YOUTUBE_CLIENT_TEST_CHANNEL_ID")

	if videos, err := c.GetChannelVideos(channelID); err != nil {
		t.Error(err)
	} else {
		for _, video := range videos {
			t.Log(video)
		}
	}
}
