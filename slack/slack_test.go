package slack

import (
	"testing"

	"github.com/rs/zerolog/log"
)

func TestConnectSlack(t *testing.T) {
	connectSlack("")
	team := GetTeam("pyspa")
	team.StartRTM(func(i int, a ...string) {
		log.Debug().Msgf("%v", a)
	})
}
