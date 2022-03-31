package consts

const (
	Long  = true
	Short = false
)

const (
	DNSPort   uint16 = 53
	Localhost        = "127.0.0.1/32"
	// InboundPassthroughIPv4SourceAddress
	// TODO (bartsmykla): add some description
	InboundPassthroughIPv4SourceAddress = "127.0.0.6/32"
)

var Flags = map[string]map[bool]string{
	// commands
	"append": {
		Long:  "--append",
		Short: "-A",
	},
	"new-chain": {
		Long:  "--new-chain",
		Short: "-N",
	},

	// parameters
	"jump": {
		Long:  "--jump",
		Short: "-j",
	},
}
