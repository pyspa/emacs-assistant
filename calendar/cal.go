package calendar

import (
	"context"
	"fmt"
	"libpyspaemacs/config"
	"strings"
	"time"

	"github.com/mopemope/emacs-module-go"
	"github.com/pkg/errors"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var (
	maxResults               = 50
	period     time.Duration = 2
)

type Calendar struct {
	config *config.Config
}

func NewCalendar(c *config.Config) *Calendar {
	return &Calendar{
		config: c,
	}
}

func (c *Calendar) Auth(ectx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ectx.Environment()
	stdlib := env.StdLib()
	gcp := NewGCPAuthWrapper()
	if _, err := gcp.Auth(c.config.Assisstant.Credential); err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed auth")
	}
	return stdlib.T(), nil
}

func (c *Calendar) RetrieveSchedules(ectx emacs.FunctionCallContext) (emacs.Value, error) {
	env := ectx.Environment()
	stdlib := env.StdLib()

	calendars, err := listToArray(env, ectx.Arg(0))
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed parse calendars")
	}
	calendars = append(calendars, env.String("primary"))
	gcp := NewGCPAuthWrapper()
	tok, err := gcp.Auth(c.config.Assisstant.Credential)
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed auth")
	}

	ctx := context.Background()
	client := gcp.Config.Client(context.Background(), tok)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return stdlib.Nil(), errors.Wrap(err, "failed init service")
	}

	now := time.Now().UTC()
	minTime := now.Format(time.RFC3339)
	maxTime := now.Add(24 * time.Hour * period).Format(time.RFC3339)
	var res []emacs.Value

	for _, ecal := range calendars {

		cal, err := env.GoString(ecal)
		if err != nil {
			return stdlib.Nil(), errors.Wrap(err, "failed convert string")
		}
		if cal == "" {
			// next
			continue
		}
		events, err := srv.Events.List(cal).ShowDeleted(false).
			SingleEvents(true).TimeMin(minTime).TimeMax(maxTime).MaxResults(int64(maxResults)).OrderBy("startTime").Do()
		if err != nil {
			return stdlib.Nil(), errors.Wrap(err, "unable to retrieve next user's events")
		}

		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}

			var sch []emacs.Value

			sch = append(sch, env.String(item.Id))
			sch = append(sch, env.String(item.Summary))
			sch = append(sch, env.String(fmt.Sprintf("<%s>", strings.Replace(date, "T", " ", 1))))
			if item.ConferenceData != nil {
				meet := fmt.Sprintf("https://meet.google.com/%s", item.ConferenceData.ConferenceId)
				sch = append(sch, env.String(meet))
			}
			list := stdlib.List(sch...)
			res = append(res, list)
		}
	}
	return stdlib.List(res...), nil
}

func listToArray(env emacs.Environment, lst emacs.Value) ([]emacs.Value, error) {

	stdlib := env.StdLib()
	var res []emacs.Value
	car := stdlib.Intern("car")
	cdr := stdlib.Intern("cdr")

	for {
		// first
		elem, err := stdlib.Funcall(car, lst)
		if err != nil {
			return nil, errors.Wrap(err, "failed call car")
		}
		if !env.GoBool(elem) {
			break
		}

		res = append(res, elem)

		next, err := stdlib.Funcall(cdr, lst)
		if err != nil {
			return nil, errors.Wrap(err, "failded call cdr")
		}
		lst = next
	}
	return res, nil

}
