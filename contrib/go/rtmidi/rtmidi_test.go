package rtmidi

import (
	"fmt"
	"log"
	"testing"
)

func ExampleCompiledAPI() {
	for _, api := range CompiledAPI() {
		log.Println("Compiled API: ", api)
	}
}

func ExampleMIDIIn_Message() {
	in, err := NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer in.Destroy()
	if err := in.OpenPort(0, "RtMidi"); err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	for {
		m, t, err := in.Message()
		if len(m) > 0 {
			log.Println(m, t, err)
		}
	}
}

func ExampleMIDIIn_SetCallback() {
	in, err := NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}
	defer in.Destroy()
	if err := in.OpenPort(0, "RtMidi"); err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	in.SetCallback(func(m MIDIIn, msg []byte, t float64) {
		log.Println(msg, t)
	})
	<-make(chan struct{})
}

//
// Tests
//

// Ensure there is at least one API available
func TestCompiledAPI(t *testing.T) {
	apis := CompiledAPI()
	if len(apis) < 1 {
		t.Errorf("Compiled API list is empty")
	}
}

// Helper to close a port when the test is complete
func closeAfter(t *testing.T, m MIDI) {
	t.Cleanup(func() {
		t.Run("close", func(t *testing.T) {
			err := m.Close()
			if err != nil {
				t.Error(err)
			}
		})
	})
}

// Tests specific to a MIDIIn port
func testInputPort(t *testing.T, m MIDIIn) {
	t.Run("ignore", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			sysex := (i & 1)
			sense := ((i >> 1) & 1)
			timing := ((i >> 2) & 1)

			k := fmt.Sprintf("%d%d%d", sysex, timing, sense)
			t.Run(k, func(t *testing.T) {
				err := m.IgnoreTypes(sysex == 1, timing == 1, sense == 1)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})

	t.Run("callback", func(t *testing.T) {
		callback := func(MIDIIn, []byte, float64) {
			// do nothing
		}
		err := m.SetCallback(callback)
		if err != nil {
			t.Fatal(err)
		}
		err = m.CancelCallback()
		if err != nil {
			t.Error(err)
		}
	})
}

// Tests specific to a MIDIOut port
func testOutputPort(t *testing.T, m MIDIOut) {
	messages := []struct {
		name  string
		bytes []byte
	}{
		{"note-on", []byte{0x90, 0x30, 0x60}},
		{"note-off", []byte{0x80, 0x30, 0x00}},
	}

	t.Run("send", func(t *testing.T) {
		for _, msg := range messages {
			t.Run(msg.name, func(t *testing.T) {
				err := m.SendMessage(msg.bytes)
				if err != nil {
					t.Error(err)
				}
			})
		}
	})
}

func testVirtualPort(m MIDI, err error) func(t *testing.T) {
	return func(t *testing.T) {
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(t, m)

		err = m.OpenVirtualPort("RtMidiVirtual")
		if err != nil {
			t.Error(err)
		}

		if testing.Short() {
			return
		}

		switch mm := m.(type) {
		case MIDIIn:
			testInputPort(t, mm)
		case MIDIOut:
			testOutputPort(t, mm)
		default:
			t.Fatalf("Unexpected port type %T", mm)
		}
	}
}

func testExistingPort(m MIDI, err error) func(t *testing.T) {
	return func(t *testing.T) {
		if err != nil {
			t.Fatal(err)
		}
		closeAfter(t, m)

		var n int
		var name string

		t.Run("count", func(t *testing.T) {
			n, err = m.PortCount()
			if err != nil {
				t.Error(err)
			}
		})

		if testing.Short() {
			return
		}

		if n < 1 {
			t.Fatal("There were zero available ports")
		}

		t.Run("name", func(t *testing.T) {
			name, err = m.PortName(0)
			if err != nil {
				t.Error(err)
			}
		})

		if name == "" {
			t.Fatal("Port name is an empty string")
		}

		t.Run("open", func(t *testing.T) {
			err = m.OpenPort(0, name)
			if err != nil {
				t.Error(err)
			}
		})

		switch mm := m.(type) {
		case MIDIIn:
			testInputPort(t, mm)
		case MIDIOut:
			testOutputPort(t, mm)
		default:
			t.Fatalf("Unexpected port type %T", mm)
		}
	}
}

// Run tests for each API discovered
func TestAPIs(t *testing.T) {
	for _, api := range CompiledAPI() {
		name := api.String()
		if name == "?" {
			name = fmt.Sprintf("RtMidiApi(%d)", int(api))
			t.Errorf("API %s is unnamed", name)
		}

		t.Run(name, func(t *testing.T) {
			t.Run("output", testExistingPort(NewMIDIOut(api, "RtMidi")))
			t.Run("input", testExistingPort(NewMIDIIn(api, "RtMidi", 1024)))
		})
	}
}

func TestDefaults(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Run("output", testExistingPort(NewMIDIOutDefault()))
		t.Run("input", testExistingPort(NewMIDIInDefault()))
	})
	t.Run("virtual", func(t *testing.T) {
		t.Run("output", testVirtualPort(NewMIDIOutDefault()))
		t.Run("input", testVirtualPort(NewMIDIInDefault()))
	})
}
