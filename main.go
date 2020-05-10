package main

// int plugin_is_GPL_compatible;
import "C"

import (
	"libpyspaemacs/speech"
	"strings"

	"github.com/mopemope/emacs-module-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	initLogger()
	emacs.Register(initModule)
}

func initModule(env emacs.Environment) {
	log.Debug().Msg("initializing ...")
	// echo
	env.RegisterFunction("pyspa/echo", echo, 1, "doc", nil)
	// speech
	env.RegisterFunction("pyspa/speech", speech.Speech, 2, "doc", nil)

	env.ProvideFeature("pyspa")
}

func echo(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	stdlib := ctx.Environment().StdLib()
	msg, err := ctx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), err
	}
	log.Info().Msg(msg)
	return stdlib.Nil(), nil
}

func initLogger() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if true {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func main() {
}
