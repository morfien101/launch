package bytepipe

type BytePipe struct {
	Ready  chan string
	closed bool
}

func New() *BytePipe {
	return &BytePipe{
		Ready:  make(chan string, 1),
		closed: false,
	}
}

func (bp *BytePipe) Write(p []byte) (n int, err error) {
	if p[len(p)-1] != '\n' {
		p = append(p, '\n')
	}

	bp.Ready <- string(p)
	return len(p), err
}

func (bp *BytePipe) Close() {
	if bp.closed {
		return
	}
	close(bp.Ready)
	bp.closed = true
}
