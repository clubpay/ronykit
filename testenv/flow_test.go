package testenv

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit/utils"
	. "github.com/smartystreets/goconvey/convey"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func TestFlow(t *testing.T) {
	Convey("Flow", t, func(c C) {
		Prepare(t, c)

		_, _ = c.Println(temporalHostPort)
		_, _ = c.Println(TemporalUI)
		b, err := flow.NewBackend(
			flow.BackendConfig{
				TaskQueue:     "kitTest",
				Namespace:     "kitTest",
				HostPort:      temporalHostPort,
				DataConverter: flow.EncryptedDataConverter(utils.RandomID(32)),
				Logger:        flow.NewZapAdapter(zap.NewNop()),
			},
		)
		c.So(err, ShouldBeNil)
		sdk := flow.NewSDK(flow.SDKConfig{
			Logger:         flow.NewZapAdapter(zap.NewNop()),
			DefaultBackend: b,
		})
		sdk.InitWithState(&State{Name: "StateName"})

		err = sdk.Start()
		c.So(err, ShouldBeNil)
		defer sdk.Stop()

		ctx := context.Background()
		wr, err := WFSelect.Execute(ctx, "Req1", flow.ExecuteWorkflowOptions{})
		c.So(err, ShouldBeNil)

		res, err := wr.Get(ctx)
		c.So(err, ShouldBeNil)
		c.So(res.T1, ShouldEqual, "t1:canceled")
		c.So(res.T2, ShouldEqual, "t2:canceled")
		c.So(res.Messages, ShouldResemble, []string{
			"hello",
			"world",
		})

		_, _ = c.Println(res)
		_, _ = c.Println("==========================================================")
		_, _ = c.Println("Workflow History Items:")
		histRes, err := sdk.GetWorkflowHistory(
			ctx,
			flow.GetWorkflowHistoryRequest{
				WorkflowID: wr.ID,
				RunID:      wr.RunID,
			},
		)
		c.So(err, ShouldBeNil)
		_, _ = c.Println(len(histRes.Events))
		_ = histRes

		for _, e := range histRes.Events {
			c.Println(e.Payload)
		}
	})
}

type Response struct {
	T1       string
	T1Err    string
	T2       string
	T2Err    string
	Messages []string
}

var WFSelect = flow.NewWorkflow(
	"Select", "",
	func(ctx *flow.WorkflowContext[string, Response, *State], req string) (*Response, error) {
		t1 := ctx.Timer(time.Second * 5)
		t2 := ctx.Timer(time.Second * 10)
		ch := flow.NewBufferedChannel[string](ctx.Context(), 10)
		ch.SendAsync("hello")
		ch.SendAsync("world")
		ch.Close()
		res := Response{}
		s := ctx.NamedSelector("select1")
		flow.SelectorAddReceive[string](s, ch, func(ch flow.Channel[string], more bool) {
			msg, ok := ch.Receive(ctx.Context())
			if ok {
				res.Messages = append(res.Messages, msg)
			}
			ctx.Log().Info("channel received", msg, ok)
		})
		flow.SelectorAddFuture(s, t1, func(f flow.Future[temporal.CanceledError]) {
			ctx.Log().Info("t1 received")
			x, err := f.Get(ctx.Context())
			if err != nil {
				res.T1Err = err.Error()
			}
			if x != nil {
				res.T1 = fmt.Sprintf("t1:%v", x)
			}
		})
		flow.SelectorAddFuture(s, t2, func(f flow.Future[temporal.CanceledError]) {
			ctx.Log().Info("t2 received")
			x, err := f.Get(ctx.Context())
			if err != nil {
				res.T2Err = err.Error()
			}
			if x != nil {
				res.T2 = fmt.Sprintf("t2:%v", x)
			}
		})

		for i := 0; i < 5; i++ {
			s.Select(ctx.Context())
		}

		name, err := ACTGetName.Execute(
			ctx.Context(), "Req1",
			flow.ExecuteActivityOptions{},
		).Get(ctx.Context())
		if err != nil {
			return nil, err
		}

		fmt.Println(*name)

		return utils.ValPtr(res), nil
	},
)

type State struct {
	Name string
}

var ACTGetName = flow.ToActivityFactory[*State, string, string](
	"GetName", "",
	func(s *State) flow.ActivityRawFunc[string, string] {
		return s.GetName
	},
)

func (s *State) GetName(ctx context.Context, req string) (*string, error) {
	out := req + " " + s.Name

	return &out, nil
}
