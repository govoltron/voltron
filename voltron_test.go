package voltron_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/govoltron/matrix"
	"github.com/govoltron/voltron"
	"github.com/govoltron/voltron/builtin/client/http"
	"github.com/govoltron/voltron/service/example"
)

var (
	ip  http.Client
	ctx = context.Background()
	vol = voltron.New(ctx)
)

func init() {
	voltron.ClientVarP(&ip, &http.ClientOptions{
		SevName:    "ip",
		Endpoints:  []matrix.Endpoint{{Addr: "114.116.209.130:8099", Weight: 100}},
		Scheme:     "http",
		Host:       "open.17paipai.cn",
		Timeout:    3000,
		RetryCount: 0,
	}, "ip")
	// voltron.ClientVarP(&ip, voltron.Discovery("ip"), "ip")
}

func TestVoltron_Run(t *testing.T) {
	kvs, err := matrix.NewEtcdStore(ctx, []string{"127.0.0.1:2379"})
	if err != nil {
		t.Errorf("NewEtcdStore failed, error is %s", err.Error())
		return
	}
	cluster, err := matrix.NewCluster(ctx, "cu4k6mg398qd", kvs)
	if err != nil {
		t.Errorf("NewMatrix failed, error is %s", err.Error())
		return
	}
	// Join the cluster
	vol.Join(cluster)

	vol.Setup(example.New(), "example service") // voltron.WithAutoReport("114.116.209.130:8099", 100, 10),
	vol.Setup(voltron.ServiceFunc(func(ctx context.Context) {
		resp, err1 := ip.Get("/_ip/", nil)
		if err1 != nil {
			fmt.Printf("e1: %s\n", err1.Error())
		} else {
			buf, err2 := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("e2: %s\n", err2.Error())
			} else {
				fmt.Printf("resp: %s\n", string(buf))
			}
		}
	}), "test run function")

	if err := vol.Run(ctx); err != nil {
		t.Errorf("%s", err.Error())
	}

	vol.Print()

}
