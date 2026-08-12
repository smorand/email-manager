package calendar

import (
	"github.com/google/uuid"
	gcal "google.golang.org/api/calendar/v3"
)

// EventInput carries the fields accepted by `cal add` / `cal update` for
// building or patching a Google Calendar event.
type EventInput struct {
	Summary         string
	Description     string
	Location        string
	Start           string // RFC3339 datetime, or "YYYY-MM-DD" when AllDay is set.
	End             string
	AllDay          bool
	TimeZone        string
	Attendees       []string // email addresses.
	Recurrence      []string // raw RRULE strings, e.g. "RRULE:FREQ=WEEKLY;COUNT=5".
	ReminderMinutes []int64  // minutes-before-event popup reminder overrides.
	ColorID         string
	Visibility      string // "default", "public", or "private".
	Busy            *bool  // true -> transparency "opaque", false -> "transparent", nil -> unset.
	ConferenceMeet  bool   // request a Google Meet link.
}

// ToEvent builds a *gcal.Event from scratch (used by `cal add`).
func (in EventInput) ToEvent() *gcal.Event {
	ev := &gcal.Event{
		Summary:     in.Summary,
		Description: in.Description,
		Location:    in.Location,
		Start:       in.buildEventDateTime(in.Start),
		End:         in.buildEventDateTime(in.End),
		ColorId:     in.ColorID,
		Visibility:  in.Visibility,
	}
	if len(in.Recurrence) > 0 {
		ev.Recurrence = in.Recurrence
	}
	if len(in.Attendees) > 0 {
		ev.Attendees = buildAttendees(in.Attendees)
	}
	if reminders := buildReminders(in.ReminderMinutes); reminders != nil {
		ev.Reminders = reminders
	}
	if in.Busy != nil {
		ev.Transparency = transparencyFor(*in.Busy)
	}
	if in.ConferenceMeet {
		ev.ConferenceData = buildMeetRequest()
	}
	return ev
}

// buildEventDateTime renders the Start/End as an all-day Date or a
// DateTime+TimeZone, depending on in.AllDay.
func (in EventInput) buildEventDateTime(value string) *gcal.EventDateTime {
	if value == "" {
		return nil
	}
	if in.AllDay {
		return &gcal.EventDateTime{Date: value}
	}
	return &gcal.EventDateTime{DateTime: value, TimeZone: in.TimeZone}
}

// buildAttendees converts a list of email addresses into Calendar API
// attendee entries.
func buildAttendees(emails []string) []*gcal.EventAttendee {
	attendees := make([]*gcal.EventAttendee, 0, len(emails))
	for _, email := range emails {
		attendees = append(attendees, &gcal.EventAttendee{Email: email})
	}
	return attendees
}

// buildMeetRequest returns a ConferenceData create request for a new Google
// Meet link. A fresh RequestId is generated on every call, per Google's
// guidance against reusing conference data across events.
func buildMeetRequest() *gcal.ConferenceData {
	return &gcal.ConferenceData{
		CreateRequest: &gcal.CreateConferenceRequest{
			RequestId:             uuid.NewString(),
			ConferenceSolutionKey: &gcal.ConferenceSolutionKey{Type: "hangoutsMeet"},
		},
	}
}

// buildReminders returns override reminders when minutes is non-empty, or nil
// when there is nothing to override (the calendar's default reminders apply,
// and the field is simply omitted).
//
// UseDefault is a plain bool with a `json:"useDefault,omitempty"` tag, so its
// Go zero value (false) is silently dropped from the request body unless
// ForceSendFields says otherwise. Without it, the Calendar API sees no
// useDefault at all alongside the overrides and rejects the request with
// "Cannot specify both default reminders and overrides at the same time.".
func buildReminders(minutes []int64) *gcal.EventReminders {
	if len(minutes) == 0 {
		return nil
	}
	overrides := make([]*gcal.EventReminder, 0, len(minutes))
	for _, m := range minutes {
		overrides = append(overrides, &gcal.EventReminder{Method: "popup", Minutes: m})
	}
	return &gcal.EventReminders{
		UseDefault:      false,
		Overrides:       overrides,
		ForceSendFields: []string{"UseDefault"},
	}
}

// transparencyFor maps the --busy/--free flag to the API's transparency enum.
func transparencyFor(busy bool) string {
	if busy {
		return "opaque"
	}
	return "transparent"
}

// Changed reports which EventInput fields were explicitly set on the command
// line, keyed by cobra flag name. ApplyPatch only mutates fields present here.
type Changed map[string]bool

// ApplyPatch mutates an existing event in place with only the fields marked
// as changed, leaving every other field untouched. Used by `cal update` so a
// partial PATCH never clobbers unrelated fields.
func ApplyPatch(ev *gcal.Event, in EventInput, changed Changed) *gcal.Event {
	if changed["summary"] {
		ev.Summary = in.Summary
	}
	if changed["description"] {
		ev.Description = in.Description
	}
	if changed["location"] {
		ev.Location = in.Location
	}
	if changed["start"] || changed["all-day"] || changed["timezone"] {
		if in.Start != "" {
			ev.Start = in.buildEventDateTime(in.Start)
		}
	}
	if changed["end"] || changed["all-day"] || changed["timezone"] {
		if in.End != "" {
			ev.End = in.buildEventDateTime(in.End)
		}
	}
	if changed["attendee"] {
		ev.Attendees = buildAttendees(in.Attendees)
	}
	if changed["recurrence"] {
		ev.Recurrence = in.Recurrence
	}
	if changed["reminder-minutes"] {
		if reminders := buildReminders(in.ReminderMinutes); reminders != nil {
			ev.Reminders = reminders
		} else {
			ev.Reminders = &gcal.EventReminders{UseDefault: true}
		}
	}
	if changed["color-id"] {
		ev.ColorId = in.ColorID
	}
	if changed["visibility"] {
		ev.Visibility = in.Visibility
	}
	if changed["busy"] || changed["free"] {
		if in.Busy != nil {
			ev.Transparency = transparencyFor(*in.Busy)
		}
	}
	if changed["meet"] && in.ConferenceMeet {
		ev.ConferenceData = buildMeetRequest()
	}
	return ev
}
