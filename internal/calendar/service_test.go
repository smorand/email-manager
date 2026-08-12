package calendar

import (
	"strings"
	"testing"

	gcal "google.golang.org/api/calendar/v3"
)

func TestValidateNotify(t *testing.T) {
	tests := []struct {
		notify  string
		wantErr bool
	}{
		{notify: "none", wantErr: false},
		{notify: "all", wantErr: false},
		{notify: "externalOnly", wantErr: false},
		{notify: "everyone", wantErr: true},
		{notify: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.notify, func(t *testing.T) {
			err := ValidateNotify(tt.notify)
			if tt.wantErr != (err != nil) {
				t.Errorf("ValidateNotify(%q) error = %v; wantErr %v", tt.notify, err, tt.wantErr)
			}
		})
	}
}

func TestEventTime(t *testing.T) {
	if got := EventTime(nil); got != "" {
		t.Errorf("EventTime(nil) = %q; want empty", got)
	}
	if got := EventTime(&gcal.EventDateTime{Date: "2026-07-14"}); got != "2026-07-14" {
		t.Errorf("EventTime all-day = %q; want 2026-07-14", got)
	}
	if got := EventTime(&gcal.EventDateTime{DateTime: "2026-07-14T10:00:00+02:00"}); got != "2026-07-14T10:00:00+02:00" {
		t.Errorf("EventTime datetime = %q", got)
	}
}

func TestMeetLink(t *testing.T) {
	if got := MeetLink(nil); got != "" {
		t.Errorf("MeetLink(nil) = %q; want empty", got)
	}
	if got := MeetLink(&gcal.Event{}); got != "" {
		t.Errorf("MeetLink(no conference data) = %q; want empty", got)
	}
	ev := &gcal.Event{
		ConferenceData: &gcal.ConferenceData{
			EntryPoints: []*gcal.EntryPoint{
				{EntryPointType: "phone", Uri: "tel:+1234"},
				{EntryPointType: "video", Uri: "https://meet.google.com/abc-defg-hij"},
			},
		},
	}
	if got := MeetLink(ev); got != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("MeetLink = %q; want the video entry point", got)
	}
}

func TestFormatEventLine(t *testing.T) {
	ev := &gcal.Event{
		Id:       "evt1",
		Summary:  "Team sync",
		Start:    &gcal.EventDateTime{DateTime: "2026-01-05T09:00:00"},
		End:      &gcal.EventDateTime{DateTime: "2026-01-05T09:30:00"},
		Location: "Room 1",
	}
	line := FormatEventLine(ev)
	for _, want := range []string{"evt1", "Team sync", "2026-01-05T09:00:00", "2026-01-05T09:30:00", "Room 1"} {
		if !strings.Contains(line, want) {
			t.Errorf("FormatEventLine output %q missing %q", line, want)
		}
	}
	if strings.Contains(line, "recurring") {
		t.Errorf("non-recurring event should not be marked recurring: %q", line)
	}

	recurring := &gcal.Event{Id: "evt2", Summary: "Standup", Recurrence: []string{"RRULE:FREQ=DAILY"}}
	if !strings.Contains(FormatEventLine(recurring), "recurring") {
		t.Errorf("recurring event should be marked as such: %q", FormatEventLine(recurring))
	}
}
