package assistant

// func TestAssistant(t *testing.T) {
// 	configDir, err := os.UserConfigDir()
// 	if err != nil {
// 		log.Error().Msg(err.Error())
// 	} else {
// 		cfgFile := filepath.Join(configDir, "pyspa-config.toml")
// 		viper.SetConfigFile(cfgFile)
// 		if err := viper.ReadInConfig(); err != nil {
// 			log.Error().Msg(err.Error())
// 		} else {
// 			log.Debug().Msgf("readed config %s", cfgFile)
// 		}
// 	}

// 	text, err := ask("今日のニュースは？", false)
// 	if err != nil {
// 		log.Fatal().Err(err).Msg("")
// 	}
// 	log.Debug().Str("text", text).Msg("")
// }
