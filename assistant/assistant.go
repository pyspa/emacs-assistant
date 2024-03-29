package assistant

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"libpyspaemacs/config"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/mopemope/emacs-module-go"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	embedded "google.golang.org/genproto/googleapis/assistant/embedded/v1alpha2"
	"google.golang.org/grpc"
)

var (
	conversationState []byte
)

type Assistant struct {
	config *config.Config
}

func NewAssistant(c *config.Config) *Assistant {
	return &Assistant{
		config: c,
	}
}

func (as *Assistant) Auth(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ctx.Environment()
	stdlib := env.StdLib()
	gcp := NewGCPAuthWrapper()
	if err := gcp.Auth(as.config.Assisstant.Credential); err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed auth")
	}
	return stdlib.T(), nil
}

func (as *Assistant) Ask(ctx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ctx.Environment()
	stdlib := env.StdLib()
	text, err := ctx.GoStringArg(0)
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "")
	}
	textOnly := false
	value := ctx.Arg(1)
	if env.GoBool(value) {
		textOnly = true
	}
	res, err := as.ask(text, textOnly)
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

func (as *Assistant) ask(text string, textOnly bool) (string, error) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	gcp := NewGCPAuthWrapper()
	if err := gcp.Auth(as.config.Assisstant.Credential); err != nil {
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
		DialogStateIn: &embedded.DialogStateIn{
			LanguageCode:      "ja-JP",
			ConversationState: nil,
			IsNewConversation: true,
		},
		DeviceConfig: &embedded.DeviceConfig{
			DeviceId:      "my-emacs",
			DeviceModelId: "emacs",
		},
		Type: &embedded.AssistConfig_TextQuery{
			TextQuery: text,
		},
		DebugConfig: &embedded.DebugConfig{
			ReturnDebugInfo: true,
		},
	}

	bufOut := make([]int16, 400)
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
		time.Sleep(time.Millisecond * 100)
		defer portaudio.Terminate()
	}

	responseText := ""
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
		//log.Info().Msgf("## %+v %+v %+v", resp.GetDebugInfo(), resp.GetDeviceAction(), resp.GetSpeechResults())
		displayText := resp.GetDialogStateOut().GetSupplementalDisplayText()

		if resp.GetDialogStateOut() != nil {
			if responseText == "" {
				responseText = displayText
			}
			if textOnly {
				if responseText == "" {
					responseText = "お役に立てそうもありません"
				}
				log.Debug().Str("responseText", responseText).Msg("")
				log.Info().Str("responseText", responseText).Msg("")
				return responseText, nil
			}
		}

		if !textOnly {
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
	}

	if responseText == "" {
		responseText = "お役に立てそうもありません"
	}
	log.Info().Str("responseText", responseText).Msg("")
	return responseText, nil
}
