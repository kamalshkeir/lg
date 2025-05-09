package lg

var (
	ss       = NewLimitedSlice[string](20)
	pub      Publisher
	topicPub = ""
	usePub   = false
	saveMem  = true
)

type Publisher interface {
	Publish(topic string, data map[string]any)
}

func SaveToMem(nbLogs int) {
	saveMem = true
	ss = NewLimitedSlice[string](nbLogs)
}

func UsePublisher(publisher Publisher, topic string) {
	usePub = true
	pub = publisher
	topicPub = topic
}

func GetLogs() *LimitedSlice[string] {
	return ss
}
