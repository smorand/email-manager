package calendar

import (
	"testing"

	gcal "google.golang.org/api/calendar/v3"
)

func TestToEvent(t *testing.T) {
	tests := []struct {
		name  string
		input EventInput
		check func(t *testing.T, ev *gcal.Event)
	}{
		{
			name: "simple datetime event",
			input: EventInput{
				Summary:  "Standup",
				Start:    "2026-01-05T09:00:00",
				End:      "2026-01-05T09:30:00",
				TimeZone: "Europe/Paris",
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Start.DateTime != "2026-01-05T09:00:00" || ev.Start.TimeZone != "Europe/Paris" {
					t.Errorf("unexpected start: %+v", ev.Start)
				}
				if ev.Start.Date != "" || ev.End.Date != "" {
					t.Errorf("expected no Date field on a datetime event, got start=%q end=%q", ev.Start.Date, ev.End.Date)
				}
			},
		},
		{
			name: "all-day event",
			input: EventInput{
				Summary: "Holiday",
				Start:   "2026-07-14",
				End:     "2026-07-15",
				AllDay:  true,
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Start.Date != "2026-07-14" || ev.End.Date != "2026-07-15" {
					t.Errorf("unexpected all-day dates: start=%q end=%q", ev.Start.Date, ev.End.Date)
				}
				if ev.Start.DateTime != "" {
					t.Errorf("expected no DateTime on all-day event, got %q", ev.Start.DateTime)
				}
			},
		},
		{
			name: "recurrence RRULE",
			input: EventInput{
				Summary:    "Weekly sync",
				Start:      "2026-01-05T09:00:00",
				End:        "2026-01-05T09:30:00",
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO;COUNT=10"},
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if len(ev.Recurrence) != 1 || ev.Recurrence[0] != "RRULE:FREQ=WEEKLY;BYDAY=MO;COUNT=10" {
					t.Errorf("unexpected recurrence: %v", ev.Recurrence)
				}
			},
		},
		{
			name: "reminder overrides",
			input: EventInput{
				Summary:         "Dentist",
				Start:           "2026-01-05T09:00:00",
				End:             "2026-01-05T09:30:00",
				ReminderMinutes: []int64{10, 30},
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Reminders == nil || ev.Reminders.UseDefault {
					t.Fatalf("expected non-default reminders, got %+v", ev.Reminders)
				}
				if len(ev.Reminders.Overrides) != 2 {
					t.Fatalf("expected 2 overrides, got %d", len(ev.Reminders.Overrides))
				}
				if ev.Reminders.Overrides[0].Minutes != 10 || ev.Reminders.Overrides[0].Method != "popup" {
					t.Errorf("unexpected override: %+v", ev.Reminders.Overrides[0])
				}
				if !containsString(ev.Reminders.ForceSendFields, "UseDefault") {
					t.Errorf("expected ForceSendFields to include UseDefault so false is actually sent, got %v", ev.Reminders.ForceSendFields)
				}
			},
		},
		{
			name: "no reminders means field omitted",
			input: EventInput{
				Summary: "No reminder override",
				Start:   "2026-01-05T09:00:00",
				End:     "2026-01-05T09:30:00",
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Reminders != nil {
					t.Errorf("expected nil Reminders when no overrides given, got %+v", ev.Reminders)
				}
			},
		},
		{
			name: "attendees",
			input: EventInput{
				Summary:   "1:1",
				Start:     "2026-01-05T09:00:00",
				End:       "2026-01-05T09:30:00",
				Attendees: []string{"a@example.com", "b@example.com"},
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if len(ev.Attendees) != 2 {
					t.Fatalf("expected 2 attendees, got %d", len(ev.Attendees))
				}
				if ev.Attendees[0].Email != "a@example.com" || ev.Attendees[1].Email != "b@example.com" {
					t.Errorf("unexpected attendees: %+v", ev.Attendees)
				}
			},
		},
		{
			name: "meet conference request",
			input: EventInput{
				Summary:        "Video call",
				Start:          "2026-01-05T09:00:00",
				End:            "2026-01-05T09:30:00",
				ConferenceMeet: true,
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.ConferenceData == nil || ev.ConferenceData.CreateRequest == nil {
					t.Fatalf("expected a conference create request, got %+v", ev.ConferenceData)
				}
				if ev.ConferenceData.CreateRequest.RequestId == "" {
					t.Errorf("expected a non-empty RequestId")
				}
				if ev.ConferenceData.CreateRequest.ConferenceSolutionKey == nil ||
					ev.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
					t.Errorf("expected hangoutsMeet solution key, got %+v", ev.ConferenceData.CreateRequest.ConferenceSolutionKey)
				}
			},
		},
		{
			name: "no meet means no conference data",
			input: EventInput{
				Summary: "Plain event",
				Start:   "2026-01-05T09:00:00",
				End:     "2026-01-05T09:30:00",
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.ConferenceData != nil {
					t.Errorf("expected nil ConferenceData, got %+v", ev.ConferenceData)
				}
			},
		},
		{
			name: "busy transparency",
			input: EventInput{
				Summary: "Focus block",
				Start:   "2026-01-05T09:00:00",
				End:     "2026-01-05T09:30:00",
				Busy:    boolPtr(true),
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Transparency != "opaque" {
					t.Errorf("expected opaque transparency, got %q", ev.Transparency)
				}
			},
		},
		{
			name: "free transparency",
			input: EventInput{
				Summary: "Optional hangout",
				Start:   "2026-01-05T09:00:00",
				End:     "2026-01-05T09:30:00",
				Busy:    boolPtr(false),
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Transparency != "transparent" {
					t.Errorf("expected transparent transparency, got %q", ev.Transparency)
				}
			},
		},
		{
			name: "unset busy leaves transparency empty",
			input: EventInput{
				Summary: "Default",
				Start:   "2026-01-05T09:00:00",
				End:     "2026-01-05T09:30:00",
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Transparency != "" {
					t.Errorf("expected empty transparency, got %q", ev.Transparency)
				}
			},
		},
		{
			name: "visibility",
			input: EventInput{
				Summary:    "Private matter",
				Start:      "2026-01-05T09:00:00",
				End:        "2026-01-05T09:30:00",
				Visibility: "private",
			},
			check: func(t *testing.T, ev *gcal.Event) {
				if ev.Visibility != "private" {
					t.Errorf("expected private visibility, got %q", ev.Visibility)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := tt.input.ToEvent()
			if ev.Summary != tt.input.Summary {
				t.Errorf("Summary = %q; want %q", ev.Summary, tt.input.Summary)
			}
			tt.check(t, ev)
		})
	}
}

func TestApplyPatch(t *testing.T) {
	base := func() *gcal.Event {
		return &gcal.Event{
			Id:          "abc123",
			Summary:     "Original summary",
			Description: "Original description",
			Location:    "Original location",
			Start:       &gcal.EventDateTime{DateTime: "2026-01-05T09:00:00", TimeZone: "Europe/Paris"},
			End:         &gcal.EventDateTime{DateTime: "2026-01-05T09:30:00", TimeZone: "Europe/Paris"},
			ColorId:     "5",
			Visibility:  "default",
		}
	}

	t.Run("only changed fields are mutated", func(t *testing.T) {
		ev := base()
		input := EventInput{Summary: "New summary", TimeZone: "Europe/Paris"}
		changed := Changed{"summary": true}

		result := ApplyPatch(ev, input, changed)

		if result.Summary != "New summary" {
			t.Errorf("Summary = %q; want %q", result.Summary, "New summary")
		}
		if result.Description != "Original description" {
			t.Errorf("Description was mutated: %q", result.Description)
		}
		if result.Location != "Original location" {
			t.Errorf("Location was mutated: %q", result.Location)
		}
		if result.ColorId != "5" {
			t.Errorf("ColorId was mutated: %q", result.ColorId)
		}
		if result.Id != "abc123" {
			t.Errorf("Id should never be touched: %q", result.Id)
		}
	})

	t.Run("start and end updated together", func(t *testing.T) {
		ev := base()
		input := EventInput{
			Start:    "2026-02-01T10:00:00",
			End:      "2026-02-01T10:30:00",
			TimeZone: "Europe/Paris",
		}
		changed := Changed{"start": true, "end": true}

		result := ApplyPatch(ev, input, changed)

		if result.Start.DateTime != "2026-02-01T10:00:00" {
			t.Errorf("Start not updated: %+v", result.Start)
		}
		if result.End.DateTime != "2026-02-01T10:30:00" {
			t.Errorf("End not updated: %+v", result.End)
		}
	})

	t.Run("reminder-minutes reset to default when empty", func(t *testing.T) {
		ev := base()
		ev.Reminders = &gcal.EventReminders{UseDefault: false, Overrides: []*gcal.EventReminder{{Method: "popup", Minutes: 10}}}
		input := EventInput{}
		changed := Changed{"reminder-minutes": true}

		result := ApplyPatch(ev, input, changed)

		if result.Reminders == nil || !result.Reminders.UseDefault {
			t.Errorf("expected reminders reset to UseDefault=true, got %+v", result.Reminders)
		}
	})

	t.Run("reminder-minutes set overrides when non-empty", func(t *testing.T) {
		ev := base()
		input := EventInput{ReminderMinutes: []int64{5}}
		changed := Changed{"reminder-minutes": true}

		result := ApplyPatch(ev, input, changed)

		if result.Reminders == nil || len(result.Reminders.Overrides) != 1 || result.Reminders.Overrides[0].Minutes != 5 {
			t.Errorf("unexpected reminders: %+v", result.Reminders)
		}
		if !containsString(result.Reminders.ForceSendFields, "UseDefault") {
			t.Errorf("expected ForceSendFields to include UseDefault, got %v", result.Reminders.ForceSendFields)
		}
	})

	t.Run("attendee replaces the full list", func(t *testing.T) {
		ev := base()
		ev.Attendees = []*gcal.EventAttendee{{Email: "old@example.com"}}
		input := EventInput{Attendees: []string{"new@example.com"}}
		changed := Changed{"attendee": true}

		result := ApplyPatch(ev, input, changed)

		if len(result.Attendees) != 1 || result.Attendees[0].Email != "new@example.com" {
			t.Errorf("unexpected attendees: %+v", result.Attendees)
		}
	})

	t.Run("meet only applied when both changed and requested", func(t *testing.T) {
		ev := base()
		input := EventInput{ConferenceMeet: false}
		changed := Changed{"meet": true}

		result := ApplyPatch(ev, input, changed)

		if result.ConferenceData != nil {
			t.Errorf("expected no conference data when ConferenceMeet is false, got %+v", result.ConferenceData)
		}
	})

	t.Run("busy/free only applied when changed", func(t *testing.T) {
		ev := base()
		input := EventInput{Busy: boolPtr(true)}
		changed := Changed{}

		result := ApplyPatch(ev, input, changed)

		if result.Transparency != "" {
			t.Errorf("expected transparency untouched, got %q", result.Transparency)
		}
	})
}

func TestFindSelfAttendee(t *testing.T) {
	ev := &gcal.Event{
		Attendees: []*gcal.EventAttendee{
			{Email: "Other@Example.com"},
			{Email: "Me@Example.com", ResponseStatus: "needsAction"},
		},
	}

	tests := []struct {
		name    string
		account string
		wantNil bool
	}{
		{name: "case-insensitive match", account: "me@example.com", wantNil: false},
		{name: "exact match", account: "Me@Example.com", wantNil: false},
		{name: "no match", account: "absent@example.com", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSelfAttendee(ev, tt.account)
			if tt.wantNil && got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
			if !tt.wantNil && got == nil {
				t.Errorf("expected a match, got nil")
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
