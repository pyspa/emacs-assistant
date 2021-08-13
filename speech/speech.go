package speech

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/mopemope/emacs-module-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

var mutex sync.Mutex

func Speech(ectx emacs.FunctionCallContext) (emacs.Value, error) {
	stdlib := ectx.Environment().StdLib()

	text, err := ectx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "")
	}
	log.Info().Msg(text)
	value := ectx.Arg(1)
	if value.IsT() {
		texts := strings.Split(text, ".\n")
		for _, text := range texts {
			ctx := context.Background()
			if err := speech(ctx, text); err != nil {
				return stdlib.Nil(), errors.Wrap(err, "failed speech")
			}
		}
		return stdlib.T(), nil
	}

	ctx := context.Background()
	if err := speech(ctx, text); err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed speech")
	}

	return stdlib.T(), nil
}

func speech(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}
	mutex.Lock()
	defer mutex.Unlock()

	max := viper.GetInt("speech.text_max")
	spText := []rune(text)
	if len(spText) > max {
		text = string(spText[:max])
	}
	cred := viper.GetString("emacs.google.credentials")
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(cred))
	if err != nil {
		return errors.Wrap(err, "failed create client")
	}
	lang := viper.GetString("speech.lang")
	rate := viper.GetFloat64("speech.speaking_rate")
	pitch := viper.GetFloat64("speech.pitch")

	req := texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		},

		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: lang,
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},

		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:    texttospeechpb.AudioEncoding_MP3,
			SpeakingRate:     rate,
			Pitch:            pitch,
			EffectsProfileId: []string{"headphone-class-device"},
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return errors.Wrap(err, "failed call tts api")
	}

	out, err := ioutil.TempFile("", "tts")
	if err != nil {
		return errors.Wrap(err, "failed create tempfile")
	}

	defer func() {
		out.Close()
		os.Remove(out.Name())
	}()

	if err := ioutil.WriteFile(out.Name(), resp.AudioContent, 0644); err != nil {
		return errors.Wrap(err, "failed write contents")
	}
	cmd := viper.GetString("speech.play_cmd")
	if err := exec.Command(cmd, out.Name()).Run(); err != nil {
		return errors.Wrap(err, "failed play")
	}

	return nil
}

func init() {
	viper.SetDefault("speech.lang", "ja-JP")
	viper.SetDefault("speech.speaking_rate", 1.4)
	viper.SetDefault("speech.pitch", 1.4)
	viper.SetDefault("speech.text_max", 1024)
	viper.SetDefault("speech.play_cmd", "mpg123")
}
