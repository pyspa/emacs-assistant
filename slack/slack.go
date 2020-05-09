package slack

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

var teams map[string]*Team

type Team struct {
	name     string
	client   *slack.Client
	channels map[string]*Channel
}

type Channel struct {
	id   string
	name string
}

func ConnectSlack(token string) error {
	client := slack.New(token)
	info, err := client.GetTeamInfo()
	if err != nil {
		return errors.Wrap(err, "failed get team info")
	}

	log.Debug().Msgf("connected team [%s]", info.Name)

	team := &Team{
		name:     info.Name,
		client:   client,
		channels: map[string]*Channel{},
	}

	channels, err := client.GetChannels(true)
	if err != nil {
		return errors.Wrap(err, "failed get team channels")
	}

	for _, channel := range channels {
		if channel.IsMember {
			team.channels[channel.Name] = &Channel{
				id:   channel.ID,
				name: channel.Name,
			}
			log.Debug().Msgf("find channel %s:%s", channel.Name, channel.ID)
		}
	}

	teams[info.Name] = team
	return nil
}

func postMessage(teamName string, channelName string, msg string) (bool, error) {
	team, ok := teams[teamName]
	if !ok {
		log.Debug().Msgf("failed find team %s", teamName)
		return false, nil
	}
	channel, ok := team.channels[channelName]
	if !ok {
		log.Debug().Msgf("failed find channel %s", channelName)
		return false, nil
	}

	if _, _, err := team.client.PostMessage(
		channel.id,
		slack.MsgOptionText(msg, false)); err != nil {
		return false, errors.Wrap(err, "failed post message")
	}

	log.Debug().
		Str("team", team.name).
		Str("channel", channel.name).
		Str("text", msg).
		Msg("post message")

	return true, nil
}

func init() {
	viper.SetDefault("slack", "true")
	teams = map[string]*Team{}
}
