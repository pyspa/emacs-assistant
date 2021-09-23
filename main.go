package main

// int plugin_is_GPL_compatible;
import "C"

import (
	"libpyspaemacs/assistant"
	"libpyspaemacs/calendar"
	"libpyspaemacs/config"
	"libpyspaemacs/slack"
	"libpyspaemacs/speech"
	"os"
	"path/filepath"
	"strings"

	"github.com/mopemope/emacs-module-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Error().Msg(err.Error())
	} else {
		cfgFile := filepath.Join(configDir, "pyspa-config.toml")
		if fileExists(cfgFile) {
			viper.SetConfigFile(cfgFile)
			if err := viper.ReadInConfig(); err != nil {
				log.Error().Msg(err.Error())
			} else {
				log.Debug().Msgf("readed config %s", cfgFile)
			}
		}
	}
	initLogger()
	emacs.Register(initModule)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func initModule(env emacs.Environment) {
	log.Debug().Msg("initializing ...")
	config := config.NewConfig()

	stdlib := env.StdLib()
	// echo
	env.RegisterFunction("pyspa/echo", echo, 1, "doc", nil)
	{
		spk := speech.NewSpeaker(config)
		// speech
		env.RegisterFunction("pyspa/speech", spk.Speech, 2, "doc", nil)
	}
	{
		// slack

		// slack init
		env.RegisterFunction("pyspa/slack-init", slack.InitSlack, 0, "doc", nil)
		// slack channels
		env.RegisterFunction("pyspa/slack-channels", slack.GetChannels, 1, "doc", nil)
		// slack post-message
		env.RegisterFunction("pyspa/slack-post-message", slack.PostMessage, 3, "doc", nil)
	}

	{
		// assisstant
		as := assistant.NewAssistant(config)
		env.RegisterFunction("pyspa/assistant-auth", as.AuthGCP, 0, "doc", nil)
		env.RegisterFunction("pyspa/assistant-ask", as.Ask, 2, "doc", nil)
	}

	{
		// calendar
		cal := calendar.NewCalendar(config)
		env.RegisterFunction("pyspa/auth-calendar", cal.Auth, 0, "doc", nil)
		env.RegisterFunction("pyspa/retrieve-schedules", cal.RetrieveSchedules, 1, "doc", nil)
	}

	stdlib.Message("loaded pyspa module")
	env.ProvideFeature("pyspa")
}

func echo(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	stdlib := ctx.Environment().StdLib()
	msg, err := ctx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), err
	}
	log.Debug().Msg(msg)
	stdlib.Message(msg)
	return stdlib.Nil(), nil
}

func initLogger() {
	debug := viper.GetBool("debug")
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
}
