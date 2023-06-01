package gate

// 初始化工具生成，后面收到添加，因为ID无规则，不能区分所在包
import (
	"github.com/jzyong/TcpMonitor/service/gate/message"
	"google.golang.org/protobuf/proto"
)

var MessageDecoder = make(map[int32]proto.Message, 1000)

func init() {
	MessageDecoder[int32(message.MID_UserLoginReq)] = &message.UserLoginRequest{}
	MessageDecoder[int32(message.MID_UserLoginRes)] = &message.UserLoginResponse{}
	MessageDecoder[int32(message.MID_ReconnectReq)] = &message.UserLoginResponse{}
	MessageDecoder[int32(message.MID_ReconnectRes)] = &message.ReconnectResponse{}
}
