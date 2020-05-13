package assistant

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/mopemope/emacs-module-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	embedded "google.golang.org/genproto/googleapis/assistant/embedded/v1alpha2"
	"google.golang.org/grpc"
)

var (
	conversationState []byte
)

func AuthGCP(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ctx.Environment()
	stdlib := env.StdLib()
	cred := viper.GetString("assistant.credentials")
	gcp := NewGCPAuthWrapper()
	if err := gcp.Auth(cred); err != nil {
		return stdlib.Nil(), errors.Wrap(err, "")
	}
	return stdlib.T(), nil
}

func Ask(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ctx.Environment()
	stdlib := env.StdLib()
	text, err := ctx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "")
	}
	res, err := ask(text, false)
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "")
	}
	return env.String(res), nil
}

func newConn(ctx context.Context) (*grpc.ClientConn, error) {

	tokenSource := gcp.Conf.TokenSource(ctx, oauthToken)
	return transport.DialGRPC(ctx,
		option.WithTokenSource(tokenSource),
		option.WithEndpoint("embeddedassistant.googleapis.com:443"),
		option.WithScopes("https://www.googleapis.com/auth/assistant-sdk-prototype"),
	)
}

func ask(text string, textOnly bool) (string, error) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	cred := viper.GetString("assistant.credentials")
	gcp := NewGCPAuthWrapper()
	if err := gcp.Auth(cred); err != nil {
		return "", errors.Wrap(err, "")
	}

	ctx := context.Background()
	runDuration := 240 * time.Second
	ctx, _ = context.WithDeadline(ctx, time.Now().Add(runDuration))
	conn, err := newConn(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to acquire connection")
	}
	defer conn.Close()

	assistant := embedded.NewEmbeddedAssistantClient(conn)
	config := &embedded.AssistConfig{
		AudioOutConfig: &embedded.AudioOutConfig{
			Encoding:         embedded.AudioOutConfig_LINEAR16,
			SampleRateHertz:  16000,
			VolumePercentage: 100,
		},
		ScreenOutConfig: &embedded.ScreenOutConfig{
			ScreenMode: embedded.ScreenOutConfig_PLAYING,
		},
		DialogStateIn: &embedded.DialogStateIn{
			LanguageCode:      "ja-JP",
			ConversationState: nil,
			IsNewConversation: true,
		},
		DeviceConfig: &embedded.DeviceConfig{
			DeviceId: "device_id",
			// DeviceModelId: "emacs-e26d1-emasc-csuxkv",
			DeviceModelId: "device_model_id",
		},
		Type: &embedded.AssistConfig_TextQuery{
			TextQuery: text,
		},
		DebugConfig: &embedded.DebugConfig{
			ReturnDebugInfo: true,
		},
	}

	bufOut := make([]int16, 800)
	streamOut, err := portaudio.OpenDefaultStream(0, 1, 16000, len(bufOut), &bufOut)
	defer func() {
		if err := streamOut.Close(); err != nil {
			//
		}
	}()
	if err = streamOut.Start(); err != nil {
		log.Panic().Err(err).Msg("")
	}

	client, err := assistant.Assist(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed assist")
	}

	log.Debug().Msgf("ask: %s", text)
	if err := client.Send(&embedded.AssistRequest{
		Type: &embedded.AssistRequest_Config{
			Config: config,
		},
	}); err != nil {
		return "", errors.Wrap(err, "failed send")
	}

	if !textOnly {
		portaudio.Initialize()
		defer portaudio.Terminate()
	}

	displayText := ""
	for {
		resp, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed recv")
		}
		if resp.EventType == embedded.AssistResponse_END_OF_UTTERANCE {
			log.Debug().Msg("END_OF_UTTERANCE")
		}

		if resp.GetDialogStateOut() != nil {
			displayText = resp.GetDialogStateOut().GetSupplementalDisplayText()

			if displayText == "" {
				displayText = "お役に立てそうもありません"
			}

			log.Debug().
				Str("display", displayText).
				Msg("")
			if textOnly {
				return displayText, nil
			}
		}

		audioOut := resp.GetAudioOut()

		if audioOut != nil {
			signal := bytes.NewBuffer(audioOut.GetAudioData())
			var err error
			for err == nil {
				err = binary.Read(signal, binary.LittleEndian, bufOut)
				if err != nil {
					break
				}

				if portErr := streamOut.Write(); portErr != nil {
					log.Error().Err(err).Msg("failed to write to audio out")
				}
			}
		}
	}
	return displayText, nil
}
