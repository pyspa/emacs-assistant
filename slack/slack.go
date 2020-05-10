package slack

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

var teams map[string]*Team

type Team struct {
	name      string
	client    *slack.Client
	channels  map[string]*Channel
	channelID map[string]*Channel
	users     map[string]*User
}

type Channel struct {
	teamName string
	id       string
	name     string
}

type User struct {
	id   string
	name string
}

func ConnectSlack(tokens ...string) ([]*Team, error) {
	var res []*Team
	for _, token := range tokens {
		t, err := connectSlack(token)
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

func connectSlack(token string) (*Team, error) {
	client := slack.New(token)
	info, err := client.GetTeamInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed get team info")
	}

	log.Debug().Msgf("connected team [%s]", info.Name)

	team := &Team{
		name:      info.Name,
		client:    client,
		channels:  map[string]*Channel{},
		channelID: map[string]*Channel{},
		users:     map[string]*User{},
	}

	channels, err := client.GetChannels(true)
	if err != nil {
		return nil, errors.Wrap(err, "failed get team channels")
	}

	for _, channel := range channels {
		if channel.IsMember {
			c := &Channel{
				id:       channel.ID,
				name:     channel.Name,
				teamName: info.Name,
			}
			team.channels[channel.Name] = c
			team.channelID[channel.ID] = c
			log.Debug().Msgf("find channel %s:%s", channel.Name, channel.ID)
		}
	}
	users, err := client.GetUsers()
	if err != nil {
		return nil, errors.Wrap(err, "failed get team users")
	}
	for _, u := range users {
		name := u.RealName
		if name == "" {
			name = u.Name
		}
		team.users[u.ID] = &User{
			id:   u.ID,
			name: name,
		}
		log.Debug().Msgf("find user %s:%s:%s", u.ID, u.Name, u.RealName)
	}

	teams[info.Name] = team
	return team, nil
}

func GetTeam(name string) *Team {
	return teams[name]
}

func (t *Team) GetChannels() []*Channel {
	var chs []*Channel
	for _, c := range t.channels {
		chs = append(chs, c)
	}
	return chs
}

type RTMCallback func(int, ...string)

func (t *Team) StartRTM(callback RTMCallback) {
	rtm := t.client.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		// log.Debug().Msg("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// type: 0
			callback(0, "hello")
		case *slack.ConnectedEvent:
			// type: 1
			log.Debug().Msgf("Infos: %v", ev.Info)
			log.Debug().Msgf("Connection counter:%v", ev.ConnectionCount)
			callback(1, "connected")

		case *slack.MessageEvent:
			// type: 2
			ch := t.channelID[ev.Channel].name
			user := t.users[ev.User].name

			log.Debug().Msgf("Message: %s ch:%s user:%s text:%s", ev.Timestamp, ch, user, ev.Text)

			callback(2, ev.Timestamp, ch, user, ev.Text)

		case *slack.PresenceChangeEvent:
			// type: 3
			log.Debug().Msgf("Presence Change: %v", ev)
			// callback(3, "")

		case *slack.LatencyReport:
			// type: 4
			log.Debug().Msgf("Current latency: %v", ev.Value)
			// callback(4, "")

		case *slack.DesktopNotificationEvent:
			// type: 5
			log.Debug().Msgf("Desktop Notification: %v", ev)
			// callback(5, "")

		case *slack.RTMError:
			// type: 6
			log.Debug().Msgf("Error: %s", ev.Error())
			callback(6, ev.Error())

		case *slack.InvalidAuthEvent:
			// type: 7
			log.Debug().Msg("Invalid credentials")
			return

		default:

			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
}

func (t *Team) PostMessage(string, channelName string, msg string) (bool, error) {
	return postMessage(t.name, channelName, msg)
}

func (c *Channel) PostMessage(msg string) (bool, error) {
	return postMessage(c.teamName, c.name, msg)
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
