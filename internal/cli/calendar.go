package cli

// This file implements the `cal` command tree: Google Calendar management
// sharing the same multi-account OAuth token as the Gmail commands in cli.go.

import (
	"fmt"
	"os"
	"strings"

	gcalsvc "email-manager/internal/calendar"

	"github.com/spf13/cobra"
	gcal "google.golang.org/api/calendar/v3"
)

// cal command flags
var (
	calAllDay     bool
	calAttendees  []string
	calCalendarID string
	calColorID    string
	calComment    string
	calEnd        string
	calFree       bool
	calBusy       bool
	calMax        int64
	calMeet       bool
	calNotify     string
	calQuery      string
	calRecurrence []string
	calReminders  []int64
	calStart      string
	calStatus     string
	calSummary    string
	calCalDesc    string
	calText       string
	calTimeZone   string
	calVisibility string
)

// Command definitions
var (
	calCmd = &cobra.Command{
		Use:   "cal",
		Short: "Manage Google Calendar events",
	}

	calCalendarsCmd = &cobra.Command{
		Use:   "calendars",
		Short: "Manage calendars",
	}

	calCalendarsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List calendars visible to this account",
		RunE:  runCalCalendarsList,
	}

	calListCmd = &cobra.Command{
		Use:   "list",
		Short: "List events in a time window",
		RunE:  runCalList,
	}

	calGetCmd = &cobra.Command{
		Use:   "get <event-id>",
		Short: "Get event details",
		Args:  cobra.ExactArgs(1),
		RunE:  runCalGet,
	}

	calInstancesCmd = &cobra.Command{
		Use:   "instances <event-id>",
		Short: "List occurrences of a recurring event",
		Args:  cobra.ExactArgs(1),
		RunE:  runCalInstances,
	}

	calAddCmd = &cobra.Command{
		Use:   "add",
		Short: "Create a calendar event",
		RunE:  runCalAdd,
	}

	calUpdateCmd = &cobra.Command{
		Use:   "update <event-id>",
		Short: "Update fields on an existing event",
		Args:  cobra.ExactArgs(1),
		RunE:  runCalUpdate,
	}

	calDeleteCmd = &cobra.Command{
		Use:   "delete <event-id>",
		Short: "Delete an event (permanent, no undo)",
		Args:  cobra.ExactArgs(1),
		RunE:  runCalDelete,
	}

	calRespondCmd = &cobra.Command{
		Use:   "respond <event-id>",
		Short: "Respond to a meeting invitation (accept/decline/tentative)",
		Args:  cobra.ExactArgs(1),
		RunE:  runCalRespond,
	}

	calQuickAddCmd = &cobra.Command{
		Use:   "quick-add",
		Short: "Create an event from natural language text",
		RunE:  runCalQuickAdd,
	}

	calFreebusyCmd = &cobra.Command{
		Use:   "freebusy",
		Short: "Query busy time slots across one or more calendars",
		RunE:  runCalFreebusy,
	}
)

// Setup functions

func setupCalendarCommands() {
	calCmd.PersistentFlags().StringVar(&calCalendarID, "calendar-id", gcalsvc.DefaultCalendarID, "Calendar ID")

	calListCmd.Flags().StringVar(&calStart, "start", "", "Window start, RFC3339 (required)")
	calListCmd.Flags().StringVar(&calEnd, "end", "", "Window end, RFC3339 (required)")
	calListCmd.Flags().StringVar(&calQuery, "query", "", "Free-text search filter")
	calListCmd.Flags().Int64Var(&calMax, "max", 50, "Maximum results")
	calListCmd.Flags().StringVar(&calTimeZone, "timezone", gcalsvc.DefaultTimeZone, "Timezone for the response")
	_ = calListCmd.MarkFlagRequired("start")
	_ = calListCmd.MarkFlagRequired("end")

	calInstancesCmd.Flags().StringVar(&calStart, "start", "", "Window start, RFC3339 (required)")
	calInstancesCmd.Flags().StringVar(&calEnd, "end", "", "Window end, RFC3339 (required)")
	calInstancesCmd.Flags().Int64Var(&calMax, "max", 50, "Maximum results")
	_ = calInstancesCmd.MarkFlagRequired("start")
	_ = calInstancesCmd.MarkFlagRequired("end")

	setupCalAddFlags()
	setupCalUpdateFlags()

	calDeleteCmd.Flags().StringVar(&calNotify, "notify", gcalsvc.NotifyNone, "Notify attendees: none|all|externalOnly")

	calRespondCmd.Flags().StringVar(&calStatus, "status", "", "Response status: accepted|declined|tentative (required)")
	calRespondCmd.Flags().StringVar(&calComment, "comment", "", "Optional response comment")
	calRespondCmd.Flags().StringVar(&calNotify, "notify", gcalsvc.NotifyNone, "Notify organizer: none|all|externalOnly")
	_ = calRespondCmd.MarkFlagRequired("status")

	calQuickAddCmd.Flags().StringVar(&calText, "text", "", "Natural language event description (required)")
	_ = calQuickAddCmd.MarkFlagRequired("text")

	calFreebusyCmd.Flags().StringArrayVar(&calAttendees, "calendar", nil, "Calendar ID to query (repeatable, required)")
	calFreebusyCmd.Flags().StringVar(&calStart, "start", "", "Window start, RFC3339 (required)")
	calFreebusyCmd.Flags().StringVar(&calEnd, "end", "", "Window end, RFC3339 (required)")
	_ = calFreebusyCmd.MarkFlagRequired("calendar")
	_ = calFreebusyCmd.MarkFlagRequired("start")
	_ = calFreebusyCmd.MarkFlagRequired("end")

	calCalendarsCmd.AddCommand(calCalendarsListCmd)

	calCmd.AddCommand(calCalendarsCmd)
	calCmd.AddCommand(calListCmd)
	calCmd.AddCommand(calGetCmd)
	calCmd.AddCommand(calInstancesCmd)
	calCmd.AddCommand(calAddCmd)
	calCmd.AddCommand(calUpdateCmd)
	calCmd.AddCommand(calDeleteCmd)
	calCmd.AddCommand(calRespondCmd)
	calCmd.AddCommand(calQuickAddCmd)
	calCmd.AddCommand(calFreebusyCmd)
}

func setupCalAddFlags() {
	calAddCmd.Flags().StringVarP(&calSummary, "summary", "s", "", "Event summary/title (required)")
	calAddCmd.Flags().StringVar(&calStart, "start", "", "Start, RFC3339 (or YYYY-MM-DD with --all-day) (required)")
	calAddCmd.Flags().StringVar(&calEnd, "end", "", "End, RFC3339 (or YYYY-MM-DD with --all-day) (required)")
	calAddCmd.Flags().BoolVar(&calAllDay, "all-day", false, "All-day event: --start/--end are YYYY-MM-DD dates")
	calAddCmd.Flags().StringVar(&calTimeZone, "timezone", gcalsvc.DefaultTimeZone, "Timezone for --start/--end")
	calAddCmd.Flags().StringVar(&calLocation, "location", "", "Event location")
	calAddCmd.Flags().StringVar(&calCalDesc, "description", "", "Event description")
	calAddCmd.Flags().StringArrayVar(&calAttendees, "attendee", nil, "Attendee email (repeatable)")
	calAddCmd.Flags().StringArrayVar(&calRecurrence, "recurrence", nil, "Raw RRULE string (repeatable), e.g. RRULE:FREQ=WEEKLY;COUNT=5")
	calAddCmd.Flags().Int64SliceVar(&calReminders, "reminder-minutes", nil, "Popup reminder minutes-before-event (repeatable)")
	calAddCmd.Flags().StringVar(&calColorID, "color-id", "", "Event color ID")
	calAddCmd.Flags().StringVar(&calVisibility, "visibility", "", "Visibility: default|public|private")
	calAddCmd.Flags().BoolVar(&calBusy, "busy", false, "Mark the event as busy (opaque)")
	calAddCmd.Flags().BoolVar(&calFree, "free", false, "Mark the event as free (transparent)")
	calAddCmd.Flags().BoolVar(&calMeet, "meet", false, "Attach a Google Meet video conference link")
	calAddCmd.Flags().StringVar(&calNotify, "notify", gcalsvc.NotifyNone, "Notify attendees: none|all|externalOnly")
	_ = calAddCmd.MarkFlagRequired("summary")
	_ = calAddCmd.MarkFlagRequired("start")
	_ = calAddCmd.MarkFlagRequired("end")
}

func setupCalUpdateFlags() {
	calUpdateCmd.Flags().StringVar(&calSummary, "summary", "", "New event summary/title")
	calUpdateCmd.Flags().StringVar(&calStart, "start", "", "New start, RFC3339 (or YYYY-MM-DD with --all-day)")
	calUpdateCmd.Flags().StringVar(&calEnd, "end", "", "New end, RFC3339 (or YYYY-MM-DD with --all-day)")
	calUpdateCmd.Flags().BoolVar(&calAllDay, "all-day", false, "Treat --start/--end as YYYY-MM-DD dates")
	calUpdateCmd.Flags().StringVar(&calTimeZone, "timezone", gcalsvc.DefaultTimeZone, "Timezone for --start/--end")
	calUpdateCmd.Flags().StringVar(&calLocation, "location", "", "New event location")
	calUpdateCmd.Flags().StringVar(&calCalDesc, "description", "", "New event description")
	calUpdateCmd.Flags().StringArrayVar(&calAttendees, "attendee", nil, "Replace attendees with this list (repeatable)")
	calUpdateCmd.Flags().StringArrayVar(&calRecurrence, "recurrence", nil, "Replace recurrence with this RRULE list (repeatable)")
	calUpdateCmd.Flags().Int64SliceVar(&calReminders, "reminder-minutes", nil, "Replace popup reminders (repeatable); pass once with no value to reset to calendar default")
	calUpdateCmd.Flags().StringVar(&calColorID, "color-id", "", "New event color ID")
	calUpdateCmd.Flags().StringVar(&calVisibility, "visibility", "", "New visibility: default|public|private")
	calUpdateCmd.Flags().BoolVar(&calBusy, "busy", false, "Mark the event as busy (opaque)")
	calUpdateCmd.Flags().BoolVar(&calFree, "free", false, "Mark the event as free (transparent)")
	calUpdateCmd.Flags().BoolVar(&calMeet, "meet", false, "Attach a new Google Meet video conference link")
	calUpdateCmd.Flags().StringVar(&calNotify, "notify", gcalsvc.NotifyNone, "Notify attendees: none|all|externalOnly")
}

// calLocation is shared between add/update, distinct from the mail "location"-less flags above.
var calLocation string

// Command handlers

func runCalCalendarsList(cmd *cobra.Command, args []string) error {
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	resp, err := service.CalendarList.List().Do()
	if err != nil {
		return fmt.Errorf("error listing calendars: %w", err)
	}

	for _, entry := range resp.Items {
		primary := ""
		if entry.Primary {
			primary = " (primary)"
		}
		fmt.Printf("%s%s\n  Summary: %s\n  Access: %s\n\n", entry.Id, primary, entry.Summary, entry.AccessRole)
	}
	return nil
}

func runCalList(cmd *cobra.Command, args []string) error {
	if err := gcalsvc.ValidateNotify(calNotify); calNotify != "" && err != nil {
		return err
	}
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	call := service.Events.List(calCalendarID).
		TimeMin(calStart).
		TimeMax(calEnd).
		SingleEvents(true).
		OrderBy("startTime").
		MaxResults(calMax).
		TimeZone(calTimeZone)
	if calQuery != "" {
		call = call.Q(calQuery)
	}

	resp, err := call.Do()
	if err != nil {
		return fmt.Errorf("error listing events: %w", err)
	}

	if len(resp.Items) == 0 {
		fmt.Fprintf(os.Stderr, "No events found\n")
		return nil
	}
	for _, ev := range resp.Items {
		fmt.Println(gcalsvc.FormatEventLine(ev))
	}
	return nil
}

func runCalGet(cmd *cobra.Command, args []string) error {
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	ev, err := service.Events.Get(calCalendarID, args[0]).Do()
	if err != nil {
		return fmt.Errorf("error getting event: %w", err)
	}

	printEventDetails(ev)
	return nil
}

func printEventDetails(ev *gcal.Event) {
	fmt.Printf("ID: %s\n", ev.Id)
	fmt.Printf("Summary: %s\n", ev.Summary)
	fmt.Printf("Start: %s\n", gcalsvc.EventTime(ev.Start))
	fmt.Printf("End: %s\n", gcalsvc.EventTime(ev.End))
	if ev.Location != "" {
		fmt.Printf("Location: %s\n", ev.Location)
	}
	if ev.Description != "" {
		fmt.Printf("Description: %s\n", ev.Description)
	}
	if ev.Organizer != nil {
		fmt.Printf("Organizer: %s\n", ev.Organizer.Email)
	}
	if len(ev.Recurrence) > 0 {
		fmt.Printf("Recurrence: %s\n", strings.Join(ev.Recurrence, ", "))
	}
	if ev.RecurringEventId != "" {
		fmt.Printf("Instance of: %s\n", ev.RecurringEventId)
	}
	if meet := gcalsvc.MeetLink(ev); meet != "" {
		fmt.Printf("Meet link: %s\n", meet)
	}
	if ev.Reminders != nil && !ev.Reminders.UseDefault && len(ev.Reminders.Overrides) > 0 {
		var mins []string
		for _, r := range ev.Reminders.Overrides {
			mins = append(mins, fmt.Sprintf("%dmin/%s", r.Minutes, r.Method))
		}
		fmt.Printf("Reminders: %s\n", strings.Join(mins, ", "))
	}
	if len(ev.Attendees) > 0 {
		fmt.Println("Attendees:")
		for _, a := range ev.Attendees {
			fmt.Printf("  - %s [%s]\n", a.Email, a.ResponseStatus)
		}
	}
}

func runCalInstances(cmd *cobra.Command, args []string) error {
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	resp, err := service.Events.Instances(calCalendarID, args[0]).
		TimeMin(calStart).
		TimeMax(calEnd).
		MaxResults(calMax).
		Do()
	if err != nil {
		return fmt.Errorf("error listing instances: %w", err)
	}

	if len(resp.Items) == 0 {
		fmt.Fprintf(os.Stderr, "No instances found in this window\n")
		return nil
	}
	for _, ev := range resp.Items {
		fmt.Println(gcalsvc.FormatEventLine(ev))
	}
	return nil
}

func runCalAdd(cmd *cobra.Command, args []string) error {
	if err := gcalsvc.ValidateNotify(calNotify); err != nil {
		return err
	}
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	if calBusy && calFree {
		return fmt.Errorf("--busy and --free are mutually exclusive")
	}

	input := calBuildInput()
	ev := input.ToEvent()

	warnIfSilentInvite(len(input.Attendees))

	call := service.Events.Insert(calCalendarID, ev).SendUpdates(calNotify)
	if calMeet {
		call = call.ConferenceDataVersion(1)
	}

	created, err := call.Do()
	if err != nil {
		return fmt.Errorf("error creating event: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Event created (ID: %s)\n", created.Id)
	if meet := gcalsvc.MeetLink(created); meet != "" {
		fmt.Fprintf(os.Stderr, "Meet link: %s\n", meet)
	}
	return nil
}

func runCalUpdate(cmd *cobra.Command, args []string) error {
	if err := gcalsvc.ValidateNotify(calNotify); err != nil {
		return err
	}
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	if calBusy && calFree {
		return fmt.Errorf("--busy and --free are mutually exclusive")
	}

	existing, err := service.Events.Get(calCalendarID, args[0]).Do()
	if err != nil {
		return fmt.Errorf("error fetching event %s: %w", args[0], err)
	}

	changed := calChangedFlags(cmd)
	input := calBuildInput()
	updated := calendarEventFromPatch(existing, input, changed)

	warnIfSilentInvite(len(input.Attendees))

	call := service.Events.Patch(calCalendarID, args[0], updated).SendUpdates(calNotify)
	if changed["meet"] {
		call = call.ConferenceDataVersion(1)
	}

	result, err := call.Do()
	if err != nil {
		return fmt.Errorf("error updating event: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Event updated (ID: %s)\n", result.Id)
	if meet := gcalsvc.MeetLink(result); meet != "" {
		fmt.Fprintf(os.Stderr, "Meet link: %s\n", meet)
	}
	return nil
}

func runCalDelete(cmd *cobra.Command, args []string) error {
	if err := gcalsvc.ValidateNotify(calNotify); err != nil {
		return err
	}
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	if err := service.Events.Delete(calCalendarID, args[0]).SendUpdates(calNotify).Do(); err != nil {
		return fmt.Errorf("error deleting event: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Event deleted\n")
	return nil
}

func runCalRespond(cmd *cobra.Command, args []string) error {
	if err := gcalsvc.ValidateNotify(calNotify); err != nil {
		return err
	}
	status, err := normalizeResponseStatus(calStatus)
	if err != nil {
		return err
	}

	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	ev, err := service.Events.Get(calCalendarID, args[0]).Do()
	if err != nil {
		return fmt.Errorf("error fetching event %s: %w", args[0], err)
	}

	self := gcalsvc.FindSelfAttendee(ev, account)
	if self == nil {
		return fmt.Errorf("account %s is not listed as an attendee on event %s", account, args[0])
	}
	self.ResponseStatus = status
	if calComment != "" {
		self.Comment = calComment
	}

	updated, err := service.Events.Patch(calCalendarID, args[0], ev).SendUpdates(calNotify).Do()
	if err != nil {
		return fmt.Errorf("error responding to event: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Responded %s to event %s\n", status, updated.Id)
	return nil
}

func runCalQuickAdd(cmd *cobra.Command, args []string) error {
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	created, err := service.Events.QuickAdd(calCalendarID, calText).Do()
	if err != nil {
		return fmt.Errorf("error creating quick-add event: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Event created (ID: %s)\n", created.Id)
	fmt.Println(gcalsvc.FormatEventLine(created))
	return nil
}

func runCalFreebusy(cmd *cobra.Command, args []string) error {
	service, err := gcalsvc.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	items := make([]*gcal.FreeBusyRequestItem, 0, len(calAttendees))
	for _, id := range calAttendees {
		items = append(items, &gcal.FreeBusyRequestItem{Id: id})
	}

	resp, err := service.Freebusy.Query(&gcal.FreeBusyRequest{
		TimeMin: calStart,
		TimeMax: calEnd,
		Items:   items,
	}).Do()
	if err != nil {
		return fmt.Errorf("error querying freebusy: %w", err)
	}

	for calID, info := range resp.Calendars {
		fmt.Printf("%s:\n", calID)
		if len(info.Errors) > 0 {
			for _, e := range info.Errors {
				fmt.Printf("  error: %s\n", e.Reason)
			}
			continue
		}
		if len(info.Busy) == 0 {
			fmt.Println("  free for the whole window")
			continue
		}
		for _, period := range info.Busy {
			fmt.Printf("  busy: %s -> %s\n", period.Start, period.End)
		}
	}
	return nil
}

// Helpers

// calBuildInput assembles an EventInput from the package-level cal* flag
// variables populated by cobra for the current command invocation.
func calBuildInput() gcalsvc.EventInput {
	var busy *bool
	switch {
	case calBusy:
		v := true
		busy = &v
	case calFree:
		v := false
		busy = &v
	}
	return gcalsvc.EventInput{
		Summary:         calSummary,
		Description:     calCalDesc,
		Location:        calLocation,
		Start:           calStart,
		End:             calEnd,
		AllDay:          calAllDay,
		TimeZone:        calTimeZone,
		Attendees:       calAttendees,
		Recurrence:      calRecurrence,
		ReminderMinutes: calReminders,
		ColorID:         calColorID,
		Visibility:      calVisibility,
		Busy:            busy,
		ConferenceMeet:  calMeet,
	}
}

// calChangedFlags returns the set of `cal update` flags explicitly passed on
// the command line, used to build a minimal PATCH.
func calChangedFlags(cmd *cobra.Command) gcalsvc.Changed {
	changed := gcalsvc.Changed{}
	names := []string{
		"summary", "description", "location", "start", "end", "all-day",
		"timezone", "attendee", "recurrence", "reminder-minutes",
		"color-id", "visibility", "busy", "free", "meet",
	}
	for _, name := range names {
		changed[name] = cmd.Flags().Changed(name)
	}
	return changed
}

// calendarEventFromPatch is a thin wrapper kept alongside the CLI handlers so
// the calendar package's ApplyPatch signature reads naturally from call sites.
func calendarEventFromPatch(existing *gcal.Event, input gcalsvc.EventInput, changed gcalsvc.Changed) *gcal.Event {
	return gcalsvc.ApplyPatch(existing, input, changed)
}

// normalizeResponseStatus validates and maps --status to the Calendar API's
// attendee responseStatus enum.
func normalizeResponseStatus(status string) (string, error) {
	switch strings.ToLower(status) {
	case "accepted", "accept":
		return "accepted", nil
	case "declined", "decline":
		return "declined", nil
	case "tentative":
		return "tentative", nil
	default:
		return "", fmt.Errorf("invalid --status %q: must be one of accepted, declined, tentative", status)
	}
}

// warnIfSilentInvite prints a guard-rail warning when attendees were passed
// but --notify is left at its safe default of "none", so the agent/user
// knows no invitation email was actually sent.
func warnIfSilentInvite(attendeeCount int) {
	if attendeeCount > 0 && calNotify == gcalsvc.NotifyNone {
		fmt.Fprintf(os.Stderr,
			"warning: %d attendee(s) set but --notify none (default): no invitation email sent.\n"+
				"         Use --notify all to actually invite them.\n", attendeeCount)
	}
}
