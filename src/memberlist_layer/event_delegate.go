package memberlist_layer

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/memberlist"
)

type EventDelegate struct {
	options EventDelegateOpts
}

type EventDelegateOpts struct {
	IncrementNodes   func()
	DecrementNodes   func()
	RemoveRaftServer func(meta NodeMeta) error
}

func NewEventDelegate(opts EventDelegateOpts) *EventDelegate {
	return &EventDelegate{
		options: opts,
	}
}

// NotifyJoin implements EventDelegate interface
func (eventDelegate *EventDelegate) NotifyJoin(node *memberlist.Node) {
	eventDelegate.options.IncrementNodes()
}

// NotifyLeave implements EventDelegate interface
func (eventDelegate *EventDelegate) NotifyLeave(node *memberlist.Node) {
	eventDelegate.options.DecrementNodes()

	var meta NodeMeta

	err := json.Unmarshal(node.Meta, &meta)

	if err != nil {
		fmt.Println("Could not get leaving node's metadata.")
		return
	}

	err = eventDelegate.options.RemoveRaftServer(meta)

	if err != nil {
		fmt.Println(err)
	}
}

// NotifyUpdate implements EventDelegate interface
func (eventDelegate *EventDelegate) NotifyUpdate(node *memberlist.Node) {
	// No-Op
}
