package replay

import (
	"fmt"
	"time"
)

type Debugger struct {
	recording    *Recording
	currentIndex int
	breakpoints  map[int]bool
	paused       bool
	speed        float64
}

func NewDebugger(recording *Recording, speed float64) *Debugger {
	return &Debugger{
		recording:    recording,
		currentIndex: 0,
		breakpoints:  make(map[int]bool),
		paused:       true,
		speed:        speed,
	}
}

func (d *Debugger) Next() (*Event, error) {
	if d.currentIndex >= len(d.recording.Events) {
		return nil, fmt.Errorf("end of recording")
	}

	event := d.recording.Events[d.currentIndex]
	d.currentIndex++

	if d.breakpoints[d.currentIndex] {
		d.paused = true
	}

	return event, nil
}

func (d *Debugger) Previous() (*Event, error) {
	if d.currentIndex <= 0 {
		return nil, fmt.Errorf("start of recording")
	}

	d.currentIndex--
	event := d.recording.Events[d.currentIndex]

	return event, nil
}

func (d *Debugger) JumpTo(index int) (*Event, error) {
	if index < 0 || index >= len(d.recording.Events) {
		return nil, fmt.Errorf("invalid index: %d", index)
	}

	d.currentIndex = index
	return d.recording.Events[index], nil
}

func (d *Debugger) JumpToTimestamp(timestamp time.Time) (*Event, error) {
	for i, event := range d.recording.Events {
		if event.Timestamp.After(timestamp) || event.Timestamp.Equal(timestamp) {
			d.currentIndex = i
			return event, nil
		}
	}

	return nil, fmt.Errorf("timestamp not found: %v", timestamp)
}

func (d *Debugger) ToggleBreakpoint(index int) {
	if d.breakpoints[index] {
		delete(d.breakpoints, index)
	} else {
		d.breakpoints[index] = true
	}
}

func (d *Debugger) SetBreakpoint(index int) {
	d.breakpoints[index] = true
}

func (d *Debugger) ClearBreakpoint(index int) {
	delete(d.breakpoints, index)
}

func (d *Debugger) ClearAllBreakpoints() {
	d.breakpoints = make(map[int]bool)
}

func (d *Debugger) Play() {
	d.paused = false
}

func (d *Debugger) Pause() {
	d.paused = true
}

func (d *Debugger) TogglePause() {
	d.paused = !d.paused
}

func (d *Debugger) IsPaused() bool {
	return d.paused
}

func (d *Debugger) SetSpeed(speed float64) {
	if speed < 0.1 {
		speed = 0.1
	}
	if speed > 10.0 {
		speed = 10.0
	}
	d.speed = speed
}

func (d *Debugger) GetSpeed() float64 {
	return d.speed
}

func (d *Debugger) GetProgress() float64 {
	if len(d.recording.Events) == 0 {
		return 0.0
	}
	return float64(d.currentIndex) / float64(len(d.recording.Events))
}

func (d *Debugger) GetCurrentEvent() *Event {
	if d.currentIndex <= 0 || d.currentIndex > len(d.recording.Events) {
		return nil
	}
	return d.recording.Events[d.currentIndex-1]
}

func (d *Debugger) GetCurrentState() *StateSnapshot {
	event := d.GetCurrentEvent()
	if event == nil {
		return nil
	}
	return event.StateSnapshot
}

func (d *Debugger) FindEventByType(eventType string, forward bool) (*Event, error) {
	start := d.currentIndex
	step := 1
	if !forward {
		step = -1
	}

	for i := start; i >= 0 && i < len(d.recording.Events); i += step {
		if d.recording.Events[i].Type == eventType {
			d.currentIndex = i
			return d.recording.Events[i], nil
		}
	}

	return nil, fmt.Errorf("event type not found: %s", eventType)
}

func (d *Debugger) FindEventByService(service string, forward bool) (*Event, error) {
	start := d.currentIndex
	step := 1
	if !forward {
		step = -1
	}

	for i := start; i >= 0 && i < len(d.recording.Events); i += step {
		if d.recording.Events[i].Service == service {
			d.currentIndex = i
			return d.recording.Events[i], nil
		}
	}

	return nil, fmt.Errorf("service not found: %s", service)
}

func (d *Debugger) GetTimeline() []TimelineEntry {
	timeline := make([]TimelineEntry, len(d.recording.Events))

	for i, event := range d.recording.Events {
		entry := TimelineEntry{
			Index:     i,
			Timestamp: event.Timestamp,
			Type:      event.Type,
			Service:   event.Service,
			Summary:   event.Summary,
			Current:   i == d.currentIndex-1,
			Breakpoint: d.breakpoints[i],
		}

		timeline[i] = entry
	}

	return timeline
}

type Recording struct {
	ID          string
	Scenario    string
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Status      string
	Events      []*Event
	Metadata    map[string]interface{}
}

type Event struct {
	ID            string
	Timestamp     time.Time
	Type          string
	Service       string
	Summary       string
	Request       interface{}
	Response      interface{}
	Duration      time.Duration
	TokensUsed    int
	CostUSD       float64
	Error         error
	StateSnapshot *StateSnapshot
}

type StateSnapshot struct {
	Timestamp time.Time
	Data      map[string]interface{}
}

type TimelineEntry struct {
	Index      int
	Timestamp  time.Time
	Type       string
	Service    string
	Summary    string
	Current    bool
	Breakpoint bool
}

func (e *Event) String() string {
	icon := "•"
	switch e.Type {
	case "http_request":
		icon = "→"
	case "http_response":
		icon = "←"
	case "error":
		icon = "✗"
	case "state_change":
		icon = "⚡"
	}

	msg := fmt.Sprintf("[%s] %s %s", e.Timestamp.Format("15:04:05.000"), icon, e.Summary)

	if e.Error != nil {
		msg += fmt.Sprintf(" (error: %v)", e.Error)
	}

	return msg
}
