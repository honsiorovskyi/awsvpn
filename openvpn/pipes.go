package openvpn

import (
	"fmt"
	"os"
)

type pipe struct {
	source *os.File
	sink   *os.File
}

func (p *pipe) Close() {
	p.sink.Close()
	p.source.Close()
}

func newTextSourcePipe(text string) (*pipe, error) {
	source, sink, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("newTextSourcePipe: %w", err)
	}

	if _, err := sink.WriteString(text); err != nil {
		return nil, fmt.Errorf("newTextSourcePipe: %w", err)
	}

	// close the sink end as soon as we wrote there
	sink.Close()

	return &pipe{source: source, sink: sink}, nil
}
