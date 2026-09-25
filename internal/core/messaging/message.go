package core_messaging

import "time"

type Header struct {
	Key   string
	Value []byte
}

type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []Header
	Time      time.Time
}
