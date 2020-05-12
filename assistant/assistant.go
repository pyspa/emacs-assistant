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

func newConn(ctx context.Context) (*grpc.ClientConn, error) {

	tokenSource := gcp.Conf.TokenSource(ctx, oauthToken)
	return transport.DialGRPC(ctx,
		option.WithTokenSource(tokenSource),
		option.WithEndpoint("embeddedassistant.googleapis.com:443"),
		option.WithScopes("https://www.googleapis.com/auth/assistant-sdk-prototype"),
	)
}

func Ask(text string) error {
	portaudio.Initialize()
	defer portaudio.Terminate()

	cred := viper.GetString("assistant.credentials")
	gcp := NewGCPAuthWrapper()
	if err := gcp.Auth(cred); err != nil {
		return errors.Wrap(err, "")
	}

	ctx := context.Background()
	runDuration := 240 * time.Second
	ctx, _ = context.WithDeadline(ctx, time.Now().Add(runDuration))
	conn, err := newConn(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to acquire connection")
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
			DeviceId:      "device_id",
			DeviceModelId: "device_model_id",
		},
		Type: &embedded.AssistConfig_TextQuery{
			TextQuery: text,
		},
	}

	bufOut := make([]int16, 799)
	streamOut, err := portaudio.OpenDefaultStream(0, 1, 16000, len(bufOut), &bufOut)
	defer func() {
		if err := streamOut.Close(); err != nil {
		}
	}()
	if err = streamOut.Start(); err != nil {
		panic(err)
	}

	client, err := assistant.Assist(ctx)
	if err != nil {
		return errors.Wrap(err, "failed assist")
	}

	log.Debug().Msgf("Ask: %s", text)
	if err := client.Send(&embedded.AssistRequest{
		Type: &embedded.AssistRequest_Config{
			Config: config,
		},
	}); err != nil {
		return errors.Wrap(err, "failed send")
	}

	for {
		resp, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed recv")
		}
		if resp.EventType == embedded.AssistResponse_END_OF_UTTERANCE {
			log.Debug().Msg("END_OF_UTTERANCE")
		}

		if resp.GetDialogStateOut() != nil {
			log.Debug().
				Str("display", resp.GetDialogStateOut().GetSupplementalDisplayText()).
				Msg("")
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
	return nil
}
