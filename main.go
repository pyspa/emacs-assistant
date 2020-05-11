package main

// int plugin_is_GPL_compatible;
import "C"

import (
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
	stdlib := env.StdLib()
	// echo
	env.RegisterFunction("pyspa/echo", echo, 1, "doc", nil)
	// speech
	env.RegisterFunction("pyspa/speech", speech.Speech, 2, "doc", nil)
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
