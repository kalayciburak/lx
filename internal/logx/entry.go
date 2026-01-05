package logx

type Level int

const (
	LevelUnknown Level = iota
	LevelTrace
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Entry struct {
	Index     int
	Raw       string
	Message   string
	Timestamp string
	Level     Level
	Fields    map[string]any
	IsJSON    bool
	IsStack   bool
	Deleted   bool
}
