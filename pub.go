package lg

var (
	ss       = NewLimitedSlice[string](20)
	pub      Publisher
	topicPub = ""
)

type Publisher interface {
	Publish(topic string, data map[string]any)
}

func SaveToMem(nbLogs int) {
	if ss == nil {
		ss = NewLimitedSlice[string](nbLogs)
	}
}

func UsePublisher(publisher Publisher, topic string) {
	pub = publisher
	topicPub = topic
}

func GetLogs() *LimitedSlice[string] {
	return ss
}
