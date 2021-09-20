package config

import "github.com/spf13/viper"

const SpeechCredentialKey = "emacs.speech.credentials"
const AssistantCredentialKey = "emacs.assistant.credentials"

type Config struct {
	Assisstant AssisstantConfig
	Speech     SpeechConfig
}

type AssisstantConfig struct {
	Credential string
}

type SpeechConfig struct {
	Credential   string
	Lang         string
	SpeakingRate float64
	Pitch        float64
	TextMax      int
	PlayCommand  []string
}

func NewConfig() *Config {
	ac := NewAssistantConfigFromEnv()
	sc := NewSpeechConfigFromEnv()
	return &Config{
		Assisstant: ac,
		Speech:     sc,
	}
}

func NewAssistantConfigFromEnv() AssisstantConfig {
	ac := AssisstantConfig{}
	cred := viper.GetString(AssistantCredentialKey)
	ac.Credential = cred
	return ac
}

func NewSpeechConfigFromEnv() SpeechConfig {
	sc := SpeechConfig{}
	cred := viper.GetString(SpeechCredentialKey)
	sc.Credential = cred
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
