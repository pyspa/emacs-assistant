package config

import "github.com/spf13/viper"

const GoogleCredentialKey = "emacs.google.credentials"

type Config struct {
	GoogleCredential string
	Speech           SpeechConfig
}

type SpeechConfig struct {
	Lang         string
	SpeakingRate float64
	Pitch        float64
	TextMax      int
	PlayCommand  []string
}

func NewConfig(gcred string) *Config {
	sc := NewSpeechConfigFromEnv()
	return &Config{
		GoogleCredential: gcred,
		Speech:           sc,
	}
}

func NewSpeechConfigFromEnv() SpeechConfig {
	sc := SpeechConfig{}
	sc.Lang = viper.GetString("speech.lang")
	sc.SpeakingRate = viper.GetFloat64("speech.speaking_rate")
	sc.Pitch = viper.GetFloat64("speech.pitch")
	sc.TextMax = viper.GetInt("speech.text_max")
	sc.PlayCommand = viper.GetStringSlice("speech.play_cmd")
	return sc
}

func init() {
	viper.SetDefault("speech.lang", "ja-JP")
	viper.SetDefault("speech.speaking_rate", 2.2)
	viper.SetDefault("speech.pitch", 2.5)
	viper.SetDefault("speech.text_max", 1024)
	viper.SetDefault("speech.play_cmd", []string{"mpg123"})
}
