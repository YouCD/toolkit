package bar

import (
	"bytes"
	"io"
	"os"
	"strings"
	"time"

	"atomicgo.dev/schedule"
	"github.com/pterm/pterm"
)

var DefaultMultiPrinter = MultiPrinter{
	printers:    []pterm.LivePrinter{},
	Writer:      os.Stdout,
	UpdateDelay: time.Millisecond * 200,
	buffers:     []*bytes.Buffer{},
	area:        pterm.DefaultArea,
}

//nolint:recvcheck
type MultiPrinter struct {
	IsActive    bool
	Writer      io.Writer
	UpdateDelay time.Duration

	printers []pterm.LivePrinter
	buffers  []*bytes.Buffer
	area     pterm.AreaPrinter
}

func (p *MultiPrinter) AddBuffer(buffer *bytes.Buffer) {
	p.buffers = append(p.buffers, buffer)
}
func (p *MultiPrinter) UpdateText(text string) {
	// pterm.Fprinto(p.Writer, s.Style.Sprint(s.currentSequence)+" "+s.MessageStyle.Sprint(s.Text))
	pterm.Fprintln(p.Writer, text)
}

// SetWriter sets the writer for the AreaPrinter.
func (p *MultiPrinter) SetWriter(writer io.Writer) {
	p.Writer = writer
}

// WithWriter returns a fork of the MultiPrinter with a new writer.
func (p MultiPrinter) WithWriter(writer io.Writer) *MultiPrinter {
	p.Writer = writer
	return &p
}

// WithUpdateDelay returns a fork of the MultiPrinter with a new update delay.
func (p MultiPrinter) WithUpdateDelay(delay time.Duration) *MultiPrinter {
	p.UpdateDelay = delay
	return &p
}

func (p *MultiPrinter) NewWriter() io.Writer {
	buf := bytes.NewBufferString("")
	p.buffers = append(p.buffers, buf)
	return buf
}

func (p *MultiPrinter) Start() (*MultiPrinter, error) {
	p.IsActive = true
	for _, printer := range p.printers {
		_, _ = printer.GenericStart()
	}

	schedule.Every(p.UpdateDelay, func() bool {
		if !p.IsActive {
			return false
		}
		p.area.Update(p.getString())
		return true
	})

	return p, nil
}

func (p *MultiPrinter) Stop() (*MultiPrinter, error) {
	p.IsActive = false
	for _, printer := range p.printers {
		_, _ = printer.GenericStop()
	}
	time.Sleep(time.Millisecond * 20)
	p.area.Update(p.getString())
	_ = p.area.Stop()

	return p, nil
}

// GenericStart runs Start, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Start instead of this in your program.
func (p MultiPrinter) GenericStart() (*pterm.LivePrinter, error) {
	p2, _ := p.Start()
	lp := pterm.LivePrinter(p2)
	return &lp, nil
}

// GenericStop runs Stop, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Stop instead of this in your program.
func (p MultiPrinter) GenericStop() (*pterm.LivePrinter, error) {
	p2, _ := p.Stop()
	lp := pterm.LivePrinter(p2)
	return &lp, nil
}

// getString returns all buffers appended and separated by a newline.
func (p *MultiPrinter) getString() string {
	var buffer bytes.Buffer
	for _, b := range p.buffers {
		s := b.String()
		s = strings.Trim(s, "\n")
		parts := strings.Split(s, "\r") // only get the last override
		s = parts[len(parts)-1]
		// check if s is empty, if so get one part before, repeat until not empty
		for s == "" {
			parts = parts[:len(parts)-1]
			if len(parts)-1 < 0 {
				break
			}
			s = parts[len(parts)-1]
		}

		s = strings.Trim(s, "\n\r")
		buffer.WriteString(s)
		buffer.WriteString("\n")
	}
	return buffer.String()
}
