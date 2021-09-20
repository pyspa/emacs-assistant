package speech

import (
	"context"
	"io/ioutil"
	"libpyspaemacs/config"
	"os"
	"os/exec"
	"strings"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/mopemope/emacs-module-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

var mutex sync.Mutex

type Speaker struct {
	config *config.Config
}

func NewSpeaker(c *config.Config) *Speaker {
	return &Speaker{
		config: c,
	}
}

func (s *Speaker) Speech(ectx emacs.FunctionCallContext) (emacs.Value, error) {
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
			if err := s.speech(ctx, text); err != nil {
				return stdlib.Nil(), errors.Wrap(err, "failed speech")
			}
		}
		return stdlib.T(), nil
	}

	ctx := context.Background()
	if err := s.speech(ctx, text); err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed speech")
	}

	return stdlib.T(), nil
}

func (s *Speaker) speech(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}
	mutex.Lock()
	defer mutex.Unlock()

	max := s.config.Speech.TextMax
	spText := []rune(text)
	if len(spText) > max {
		text = string(spText[:max])
	}
	cred := s.config.Speech.Credential
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(cred))
	if err != nil {
		return errors.Wrap(err, "failed create client")
	}
	lang := s.config.Speech.Lang
	rate := s.config.Speech.SpeakingRate
	pitch := s.config.Speech.Pitch

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

	out, err := ioutil.TempFile("", "emacs.tts")
	if err != nil {
		return errors.Wrapf(err, "failed create tempfile")
	}

	defer func() {
		out.Close()
		os.Remove(out.Name())
	}()

	if err := ioutil.WriteFile(out.Name(), resp.AudioContent, 0644); err != nil {
		return errors.Wrap(err, "failed write contents")
	}
	cmds := s.config.Speech.PlayCommand
	cmd, cmds := cmds[0], cmds[1:]
	if err := exec.Command(cmd, append(cmds, out.Name())...).Run(); err != nil {
		return errors.Wrap(err, "failed play")
	}

	return nil
}
