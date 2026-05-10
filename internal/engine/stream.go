package engine

type ChannelSink struct {
	Events chan<- Event
}

func (s ChannelSink) Emit(event Event) {
	if s.Events != nil {
		s.Events <- event
	}
}
