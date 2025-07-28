package cwmetrics

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Datum struct {
	Name       string
	Value      float64
	Unit       types.StandardUnit
	Dimensions map[string]string
}

type Client struct {
	svc      *cloudwatch.Client
	buffer   map[string][]Datum
	mu       sync.Mutex
	interval time.Duration
}

func New(ctx context.Context, interval time.Duration, region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &Client{
		svc:      cloudwatch.NewFromConfig(cfg),
		buffer:   make(map[string][]Datum),
		interval: interval,
	}, nil
}

func (c *Client) Add(namespace, name string, value float64, unit types.StandardUnit, dims map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buffer[namespace] = append(c.buffer[namespace], Datum{
		Name:       name,
		Value:      value,
		Unit:       unit,
		Dimensions: dims,
	})
}

func (c *Client) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.Flush(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (c *Client) Flush(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for ns, list := range c.buffer {
		var batch []types.MetricDatum
		for _, d := range list {
			var dims []types.Dimension
			for k, v := range d.Dimensions {
				dims = append(dims, types.Dimension{
					Name:  aws.String(k),
					Value: aws.String(v),
				})
			}
			batch = append(batch, types.MetricDatum{
				MetricName: aws.String(d.Name),
				Value:      aws.Float64(d.Value),
				Unit:       d.Unit,
				Dimensions: dims,
				Timestamp:  aws.Time(now),
			})
		}
		for i := 0; i < len(batch); i += 20 {
			end := i + 20
			if end > len(batch) {
				end = len(batch)
			}
			c.svc.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String(ns),
				MetricData: batch[i:end],
			})
		}
		c.buffer[ns] = nil
	}
}
