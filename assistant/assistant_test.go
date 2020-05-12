package assistant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func TestAssistant(t *testing.T) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Error().Msg(err.Error())
	} else {
		cfgFile := filepath.Join(configDir, "pyspa-config.toml")
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Error().Msg(err.Error())
		} else {
			log.Debug().Msgf("readed config %s", cfgFile)
		}
	}

	if err := Ask("8x14は？"); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
