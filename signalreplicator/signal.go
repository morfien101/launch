// signalreplicator is used to replicate signals caught by the process manager to all registered
// processes

package signalreplicator

import (
	"fmt"
	"os"
	"sync"
)

var (
	globalReplicator *replicator
)

func init() {
	globalReplicator = new()
	go globalReplicator.listen()
}

// Register is used to add a channel to the replicator
func Register(ch chan os.Signal) {
	globalReplicator.register(ch)
}

// Remove the given channel from the replicator
func Remove(ch chan os.Signal) {
	globalReplicator.remove(ch)
}

// Send will take a signal that needs to be replicated
func Send(s os.Signal) {
	globalReplicator.input <- s
}

type replicator struct {
	sync.RWMutex
	signalChannels map[string]chan os.Signal
	input          chan os.Signal
}

func new() *replicator {
	return &replicator{
		signalChannels: make(map[string]chan os.Signal),
		input:          make(chan os.Signal, 1),
	}
}

func chanMemLocation(c chan os.Signal) string {
	return fmt.Sprintf("%p", c)
}

func (r *replicator) register(ch chan os.Signal) {
	id := chanMemLocation(ch)
	r.Lock()
	r.signalChannels[id] = ch
	r.Unlock()
}

func (r *replicator) remove(ch chan os.Signal) {
	id := chanMemLocation(ch)
	r.Lock()
	delete(r.signalChannels, id)
	r.Unlock()
}

func (r *replicator) listen() {
	for s := range r.input {
		for _, procChan := range r.signalChannels {
			procChan <- s
		}
	}
}
