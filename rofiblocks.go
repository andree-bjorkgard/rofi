package rofi

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Model struct {
	Message     string
	Overlay     string
	Prompt      string
	Input       string
	ActiveEntry int

	Options []Option
}

type blockOption struct {
	Text string `json:"text,omitempty"`
	Icon string `json:"icon,omitempty"`
	Data string `json:"data,omitempty"`

	Urgent    bool `json:"urgent,omitempty"`
	Highlight bool `json:"highlight,omitempty"`
	Markup    bool `json:"markup,omitempty"`
}

type blockModel struct {
	Message     string `json:"message"`
	Overlay     string `json:"overlay"`
	Prompt      string `json:"prompt"`
	Input       string `json:"input"`
	InputAction string `json:"input action,omitempty"`
	EventFormat string `json:"event format,omitempty"`
	ActiveEntry int    `json:"active entry,omitempty"`

	Lines []blockOption `json:"lines,omitempty"`
}

type eventName string

const (
	eventNameSelectedEntry    eventName = "SELECT_ENTRY"
	eventNameCustomEntry      eventName = "ACTIVE_ENTRY"
	eventNameCustomEntryIndex eventName = "CUSTOM_KEY"
)

func (en eventName) IsValid() bool {
	switch en {
	case eventNameSelectedEntry, eventNameCustomEntry, eventNameCustomEntryIndex:
		return true
	}
	return false
}

func (e eventName) String() string {
	return string(e)
}

type event struct {
	Name  eventName `json:"name"`
	Value string    `json:"value"`
	Index string    `json:"index"`
}

func (e event) isValid() bool {
	if !e.Name.IsValid() {
		return false
	}

	if (e.Name == eventNameCustomEntry || e.Name == eventNameSelectedEntry) && e.Value == "" {
		return false
	}
	i, _ := strconv.Atoi(e.Index)
	if e.Name == eventNameCustomEntryIndex && i < 1 {
		return false
	}

	return true
}

var eventFormat string = "{\"index\":\"{{value_escaped}}\",\"name\":\"{{name_enum}}\",\"value\":\"{{data}}\"}"

func NewRofiBlock() (Model, <-chan Value) {
	ch := make(chan Value)

	go broadcastEvents(ch)
	return Model{}, ch
}

func mapOptions(opts []Option) []blockOption {
	//bos := make([]blockOption, len(opts))
	var bos []blockOption
	for _, o := range opts {
		if o.Label == "" {
			if verbosity >= 5 {
				log.Println("Option was empty")
			}
			continue

		} else if len(o.Cmds) < 1 {
			log.Println("Can't print options with no commands")
			continue
		}

		label := o.Label
		if o.Category != "" {
			separator := " "
			if o.IsMultiline {
				separator = "\r"
			}

			label = fmt.Sprintf("%s%s%s", label, separator, o.Category)
		}

		bos = append(bos, blockOption{
			Icon:      o.Icon,
			Text:      label,
			Data:      strings.Join(append([]string{o.Value}, o.Cmds...), "||"),
			Markup:    o.UseMarkup,
			Urgent:    o.IsUrgent,
			Highlight: o.IsHighlighted,
		})
	}

	return bos
}

// Using spread to make passing selected index optional... Only cares about the first value
func (m *Model) Render(i ...int) {
	data := blockModel{
		Message:     m.Message,
		Overlay:     m.Overlay,
		Prompt:      m.Prompt,
		Input:       m.Input,
		EventFormat: eventFormat,
		Lines:       mapOptions(m.Options),
	}

	if len(i) > 0 {
		data.ActiveEntry = i[0]
	}
	j, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("rofi.Render: could not marshal: %s\n", err)
	}

	fmt.Println(string(j))
}

func getValue(eventValue string, index int) Value {
	val := Value{}
	output := strings.Split(eventValue, "||")
	if len(output) < 2 {
		log.Fatalf("Invalid event value. Needs to have a value and at least one command: %s\n", eventValue)
	}

	val.Value = output[0]

	cmds := output[1:]
	if len(cmds) <= index {
		log.Printf("Index %d did not result in a valid command. Selecting first command", index)
		index = 0
	}
	val.Cmd = cmds[index]

	return val
}

func broadcastEvents(ch chan<- Value) {
	dec := json.NewDecoder(os.Stdin)
	for {
		index := 0
		var ev event
		if err := dec.Decode(&ev); err != nil {
			log.Fatalf("rofi.broadcastEvents: Could not decode event: %s\n", err)
		}

		if !ev.isValid() {
			log.Printf("rofi.broadcastEvents: event was not valid: %#v\n", ev)
			continue
		}

		if ev.Name == eventNameCustomEntry {
			var indexEv event
			if err := dec.Decode(&indexEv); err != nil {
				log.Fatalf("rofi.broadcastEvents: Could not decode event: %s\n", err)
			}

			if indexEv.Name != eventNameCustomEntryIndex && !indexEv.isValid() {
				log.Printf("rofi.broadcastEvents: event with index was not valid: %#v\n", indexEv)
				continue
			}

			var err error
			index, err = strconv.Atoi(indexEv.Index)
			if err != nil {
				log.Printf("rofi.broadcastEvents: event with index could not be converted: %s\n", err)
			}
		}

		ch <- getValue(ev.Value, index)
	}
}
