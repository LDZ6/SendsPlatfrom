package boBing

import (
	"context"
	"encoding/base64"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	boBingPb "platform/idl/pb/boBing"
	"platform/utils"
	"testing"
)

var client boBingPb.BoBingServiceClient

func init() {
	addr := "127.0.0.1:10003"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err.Error())
	}
	client = boBingPb.NewBoBingServiceClient(conn)
}

func TestGetCount(t *testing.T) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
		"open_id", "oHCMZuK5qY75SNbZ7Kbfk6mbfDY0")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingGetCount(ctx, &emptypb.Empty{})
	logrus.Info(resp)
	logrus.Info(err)
}
func TestZhuanFa(t *testing.T) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
		"open_id", "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingRetransmission(ctx, &emptypb.Empty{})
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc = 今日增加投掷次数已达上限" {
			logrus.Info("成功")
			return
		}
		logrus.Info(err)
	}
	logrus.Info(resp)
}
func TestBoBingTianXuan(t *testing.T) {
	resp, err := client.BoBingTianXuan(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp)
}

func TestBoBingDayInit(t *testing.T) {
	req := boBingPb.BoBingInitRequest{
		StuNum:   "1919113041",
		NickName: "小G",
		OpenId:   "oHCMZuK5qY75SNbZ7Kbfk6mbfDY0",
	}
	_, err := client.BoBingDayInit(context.Background(), &req)
	if err != nil {
		logrus.Info(err)
	}

}

func TestBoBingPublish(t *testing.T) {
	req := boBingPb.BoBingKeyRequest{Openid: "oHCMZuIq0kgbnjgCUHXDbyvRDcdI"}
	resp, err := client.BoBingKey(context.Background(), &req)
	if err != nil {
		logrus.Info(err)
	}
	ase := utils.NewEncryption()
	ase.SetKey(resp.Key)
	points := ase.AesEncoding("16")
	data := []byte(`[{"position":{"x":-2.160548448562622,"y":-3.336374044418335,"z":-2.208646297454834},"quaternion":[0.7066990733146667,-0.693014919757843,-0.11863945424556732,0.0789402648806572]},{"position":{"x":3.687497138977051,"y":-3.247858762741089,"z":-1.0259438753128052},"quaternion":[-0.4946497976779938,0.3841744363307953,0.5224217176437378,-0.5786253809928894]},{"position":{"x":-2.8032286167144775,"y":-3.3492422103881836,"z":-0.11570592224597931},"quaternion":[-0.612418532371521,0.3552863597869873,0.33561211824417114,0.621353030204773]},{"position":{"x":2.3508834838867188,"y":-3.364384889602661,"z":-0.1409396231174469},"quaternion":[-0.19286783039569855,0.18030987679958344,-0.6714792847633362,0.6923915147781372]},{"position":{"x":2.505459785461426,"y":-3.3403878211975098,"z":-1.6954561471939087},"quaternion":[0.24121619760990143,-0.2175920009613037,-0.6637308597564697,0.6737432479858398]},{"position":{"x":2.2128584384918213,"y":-3.298159122467041,"z":-2.661964178085327},"quaternion":[-0.9451376795768738,-0.02570001594722271,-0.3195834755897522,0.06261748820543289]}]`)
	check := ase.AesEncoding(string(data))
	reqP := boBingPb.BoBingPublishRequest{
		Flag:     points,
		Check:    check,
		StuNum:   "",
		NickName: "【",
		OpenId:   "oHCMZuIq0kgbnjgCUHXDbyvRDcdI",
	}
	respP, err := client.BoBingPublish(context.Background(), &reqP)
	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc = 跳猴" {
			logrus.Info(1)
			return
		}
		logrus.Info(err)
		logrus.Info(2)
		return
	}
	logrus.Info(respP)
	//md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
	//	"open_id", "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8")
	//ctx := metadata.NewOutgoingContext(context.Background(), md)
	//reqC := boBingPb.BoBingBroadcastCheckRequest{
	//	Ciphertext: respP.Ciphertext,
	//	OpenId:     "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8",
	//}
	//_, err = client.BoBingBroadcastCheck(ctx, &reqC)
	//logrus.Info(err)
}

func TestGetDayRank(t *testing.T) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
		"open_id", "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingDayRank(ctx, &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp)
}

func TestRank(t *testing.T) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
		"open_id", "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingToTalTen(ctx, &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp)
}

func TestRecord(t *testing.T) {
	md := metadata.Pairs("nick_name", base64.StdEncoding.EncodeToString([]byte("橘子酒")),
		"open_id", "oHCMZuK5qY75SNbZ7Kbfk6mbfDY0")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingRecord(ctx, &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp.Maps)
}

func TestBlacklist(t *testing.T) {
	md := metadata.Pairs(
		"open_id", "oHCMZuK5qY75SNbZ7Kbfk6mbfDY0")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.BoBingBlacklist(ctx, &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp)
}

func TestNumber(t *testing.T) {
	resp, err := client.BoBingGetNumber(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Info(err)
	}
	logrus.Info(resp)
}
