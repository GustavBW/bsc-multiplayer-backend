package meta

type RuntimeMode string

const (
	RUNTIME_MODE_DEV  RuntimeMode = "dev"
	RUNTIME_MODE_PROD RuntimeMode = "prod"
	RUNTIME_MODE_TOOL RuntimeMode = "tools"
)

type MessageEncoding string

const (
	MESSAGE_ENCODING_BASE16 MessageEncoding = "base16"
	MESSAGE_ENCODING_BASE64 MessageEncoding = "base64"
	MESSAGE_ENCODING_BINARY MessageEncoding = "binary"
)

type RuntimeConfiguration struct {
	Mode     RuntimeMode
	Encoding MessageEncoding
}

func NewRuntimeConfiguration(mode RuntimeMode, encoding MessageEncoding) *RuntimeConfiguration {
	return &RuntimeConfiguration{
		Mode:     mode,
		Encoding: encoding,
	}
}
