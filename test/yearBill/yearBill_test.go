package yearBill

import (
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	yearBillPb "platform/idl/pb/yearBill"
	"testing"
)

var client yearBillPb.YearBillServiceClient

func init() {
	addr := "127.0.0.1:10005"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err.Error())
	}
	client = yearBillPb.NewYearBillServiceClient(conn)
}

func TestGetCertificate(t *testing.T) {
	req := yearBillPb.GetCertificateRequest{Openid: "oHCMZuJVYI3k1NrgjGaFxZ3a5pk8", StuNum: "2125102013"}
	resp, err := client.GetCertificate(context.Background(), &req)
	logrus.Info(resp)
	logrus.Info(err)
	req2 := yearBillPb.PayDataInitRequest{HallTicket: resp.HallTicket, StuNum: "2125102013"}
	_, err = client.PayDataInit(context.Background(), &req2)
	req3 := yearBillPb.LearnDataInitRequest{
		StuNum:       "2125102013",
		GsSession:    resp.GsSession,
		Emaphome_WEU: resp.Emaphome_WEU,
	}
	_, err = client.LearnDataInit(context.Background(), &req3)
	//req3 := yearBillPb.BookDataInitRequest{
	//	StuNum:      "2295141047",
	//	JsSessionid: resp.JsSessionId,
	//}
	//resp3, err := client.BookDataInit(context.Background(), &req3)
	logrus.Info(err)
}

func TestGetData(t *testing.T) {
	req := yearBillPb.GetLearnDataRequest{
		StuNum: "2295141047",
	}
	resp, err := client.GetLearnData(context.Background(), &req)
	logrus.Info(err)
	logrus.Info(resp)
	req2 := yearBillPb.GetPayDataRequest{StuNum: "2295141047"}
	resp2, err := client.GetPayData(context.Background(), &req2)
	logrus.Info(err)
	logrus.Info(resp2)
}
