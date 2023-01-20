package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	streamingclient "cd.splunkdev.com/abberline/signalfx-streaming-client"
	"cd.splunkdev.com/abberline/signalfx-streaming-client/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/signalfx/signalfx-go/signalflow"
)

func main() {

	// metrics exposed
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+strconv.Itoa(9000), nil)
	}()

	flow := ""
	resolutionInSeconds := int64(10)

	sfxclient, _ := signalflow.NewClient(
		signalflow.StreamURLForRealm(os.Getenv("SFX_REALM")),
		signalflow.AccessToken(os.Getenv("SFX_TOKEN")),
	)

	allowedLevel := promlog.AllowedLevel{}
	allowedLevel.Set("debug")

	promlogConfig := promlog.Config{
		Level: &allowedLevel,
	}

	client, _ := streamingclient.NewClient(sfxclient, "testclient", streamingclient.Logger(log.ConfigureLogger(&promlogConfig)))

	for {
		ctx := context.Background()
		ruleCtx := context.WithValue(ctx, "rule", "testflow")
		ruleCtx, cancel := context.WithCancel(ruleCtx)
		streamChn := make(chan streamingclient.Metrics)
		go client.StreamChannel(cancel, "testflow", flow, resolutionInSeconds, time.Now(), time.Now().Unix()-180, time.Now().Unix()-60, streamChn, 0, 60)
		count := 0
	ruleLifecycleLoop:
		for {
			select {
			case <-ruleCtx.Done():
				fmt.Println(ruleCtx.Value("rule").(string) + " is done")
				close(streamChn)
				break ruleLifecycleLoop
			case metrics := <-streamChn:
				for _, m := range metrics {
					for _, n := range m {
						count += len(n.DimensionInfo)
					}
				}
			}
		}
		fmt.Printf("Got %d metrics\n\n", count)
	}
}
