// Package calendar provides Google Calendar API service functionality.
// Named 'calendar' but the API client is imported as 'gcal' throughout to
// avoid any ambiguity with this package's own name.
package calendar

import (
	"context"
	"fmt"
	"strings"

	"email-manager/pkg/auth"

	gcal "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// DefaultCalendarID is the calendar ID for the authenticated user's primary
// calendar, accepted everywhere the Google Calendar API expects a calendar ID.
const DefaultCalendarID = "primary"

// DefaultTimeZone is used when no --timezone is passed, consistent with
// outlook-tool's calendar defaults.
const DefaultTimeZone = "Europe/Paris"

// NotifyNone, NotifyAll, and NotifyExternalOnly are the accepted values for
// the Google Calendar API's sendUpdates parameter.
const (
	NotifyNone         = "none"
	NotifyAll          = "all"
	NotifyExternalOnly = "externalOnly"
)

// GetService returns a Calendar service instance for the given account,
// reusing the same OAuth2 token as the Gmail and Contacts services.
func GetService(ctx context.Context, account string) (*gcal.Service, error) {
	client, err := auth.GetClient(ctx, account)
	if err != nil {
		return nil, err
	}

	service, err := gcal.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Calendar service: %w", err)
	}

	return service, nil
}

// ValidateNotify checks that a --notify value is one of the values accepted
// by the Google Calendar API's sendUpdates parameter.
func ValidateNotify(notify string) error {
	switch notify {
	case NotifyNone, NotifyAll, NotifyExternalOnly:
		return nil
	default:
		return fmt.Errorf("invalid --notify value %q: must be one of none, all, externalOnly", notify)
	}
}

// EventTime returns the human-readable time of an EventDateTime: the date for
// all-day events, or the dateTime (with timezone, if present) otherwise.
func EventTime(dt *gcal.EventDateTime) string {
	if dt == nil {
		return ""
	}
	if dt.Date != "" {
		return dt.Date
	}
	return dt.DateTime
}

// FormatEventLine renders a single-line summary of an event: id, summary,
// start/end, location, and a recurrence marker, used by `cal list` and
// `cal instances`.
func FormatEventLine(ev *gcal.Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s", ev.Id, ev.Summary)
	start := EventTime(ev.Start)
	end := EventTime(ev.End)
	if start != "" || end != "" {
		fmt.Fprintf(&b, "  [%s -> %s]", start, end)
	}
	if ev.Location != "" {
		fmt.Fprintf(&b, "  @ %s", ev.Location)
	}
	if len(ev.Recurrence) > 0 || ev.RecurringEventId != "" {
		b.WriteString("  (recurring)")
	}
	if MeetLink(ev) != "" {
		b.WriteString("  [meet]")
	}
	return b.String()
}

// MeetLink returns the Google Meet join URL for an event, if any conference
// data with entry points is present. Returns an empty string otherwise
// (including while a --meet conference request is still pending).
func MeetLink(ev *gcal.Event) string {
	if ev == nil || ev.ConferenceData == nil {
		return ""
	}
	for _, ep := range ev.ConferenceData.EntryPoints {
		if ep.EntryPointType == "video" && ep.Uri != "" {
			return ep.Uri
		}
	}
	return ""
}

// FindSelfAttendee returns the attendee entry matching the given account
// email (case-insensitive), or nil if the account is not listed as an
// attendee on the event (e.g. it is the organizer with no self-attendee
// entry, or the account was never invited).
func FindSelfAttendee(ev *gcal.Event, account string) *gcal.EventAttendee {
	for _, a := range ev.Attendees {
		if strings.EqualFold(a.Email, account) {
			return a
		}
	}
	return nil
}
