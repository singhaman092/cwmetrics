package main

import (
	"context"
	"time"

	"github.com/singhaman092/cwmetrics"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func main() {
	ctx := context.Background()
	client, _ := cwmetrics.New(ctx, time.Minute)
	client.Start(ctx)
	client.Add("App/Env", "LoginCount", 1, types.StandardUnitCount, map[string]string{"route": "/login"})
	time.Sleep(2 * time.Minute)
	client.Flush(ctx)
}
