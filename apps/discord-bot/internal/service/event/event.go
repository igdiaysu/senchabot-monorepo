package event

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/senchabot-opensource/monorepo/apps/discord-bot/client"
	"github.com/senchabot-opensource/monorepo/apps/discord-bot/internal/helpers"
)

func CreateLiveStreamScheduledEvent(s *discordgo.Session, msgContent string, msgEmbeds []*discordgo.MessageEmbed, guildId string, wg *sync.WaitGroup) {
	defer wg.Done()

	url := helpers.GetURL("twitch.tv", msgContent)
	if url == "" && len(msgEmbeds) > 0 {
		url = msgEmbeds[0].URL
	}

	username := helpers.ParseTwitchUsernameURLParam(url)
	if url == "" || username == "" {
		return
	}

	wg.Add(1)

	events, err := s.GuildScheduledEvents(guildId, false)
	if err != nil {
		fmt.Println("s.GuildScheduledEvents")
	}
	for _, e := range events {
		if e.Creator.Bot && e.EntityMetadata.Location == url {
			return
		}
	}

	startingTime := time.Now().Add(2 * time.Minute)
	endingTime := startingTime.Add(16 * time.Hour)

	scheduledEvent, err := s.GuildScheduledEventCreate(guildId, &discordgo.GuildScheduledEventParams{
		Name:               username + " is live on Twitch!",
		ScheduledStartTime: &startingTime,
		ScheduledEndTime:   &endingTime,
		EntityType:         discordgo.GuildScheduledEventEntityTypeExternal,
		EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
			Location: url,
		},
		PrivacyLevel: discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
	})
	if err != nil {
		log.Printf("Error while creating scheduled event: %v", err)
		wg.Done()
		return
	}

	fmt.Println("Created scheduled event: ", scheduledEvent.Name)
	wg.Done()
}

func CheckLiveStreamScheduledEvents(s *discordgo.Session) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	var twitchUsername string

	for range ticker.C {
		for _, guild := range s.State.Guilds {
			events, err := s.GuildScheduledEvents(guild.ID, false)
			if err != nil {
				fmt.Println("s.GuildScheduledEvents")
			}

			for _, e := range events {
				if !e.Creator.Bot {
					return
				}

				twitchUsername = helpers.ParseTwitchUsernameURLParam(e.EntityMetadata.Location)
				isLive, streamTitle := client.CheckTwitchStreamStatus(twitchUsername)
				if len(streamTitle) > 100 {
					streamTitle = streamTitle[0:90]
				}
				if isLive {
					if e.Name != streamTitle {
						_, err = s.GuildScheduledEventEdit(e.GuildID, e.ID, &discordgo.GuildScheduledEventParams{
							Name: streamTitle,
						})
						if err != nil {
							log.Printf("Error while updating scheduledevent: %v", err)
						}
					}
				}

				if !isLive {
					err := s.GuildScheduledEventDelete(e.GuildID, e.ID)
					if err != nil {
						log.Printf("Error deleting scheduled event: %v", err)
					}
				}
			}
		}
	}
}
