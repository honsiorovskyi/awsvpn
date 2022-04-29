package awsvpn

const (
	ErrFatal int = iota
	ErrNonFatal

	CmdConnect
	CmdDisconnect
	CmdShutdown

	StatusDisconnected
	StatusConnecting
	StatusConnected
	StatusDisconnecting
)

type Message struct {
	typ int
	err error
}

func (m Message) Type() int {
	return m.typ
}

func (m Message) Error() error {
	return m.err
}

func FatalErr(err error) Message {
	return Message{ErrFatal, err}
}

func NonFatalErr(err error) Message {
	return Message{ErrNonFatal, err}
}

func Command(cmd int) Message {
	return Message{cmd, nil}
}

func Status(st int) Message {
	return Message{st, nil}
}
