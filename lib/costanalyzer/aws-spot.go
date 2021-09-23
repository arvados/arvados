// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package costanalyzer

import (
	"fmt"
	//	"strings"
	"sort"
	"strconv"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func parseTime(layout, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return &t
}

type SpotPriceHistory []*ec2.SpotPrice

func (s SpotPriceHistory) Len() int           { return len(s) }
func (s SpotPriceHistory) Less(i, j int) bool { return s[i].Timestamp.Before(*s[j].Timestamp) }
func (s SpotPriceHistory) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func CalculateAWSSpotPrice(container arvados.Container, size string) (float64, error) {
	// FIXME hardcoded region, does this matter?
	svc := ec2.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
		//		LogLevel: aws.LogLevel(aws.LogDebugWithHTTPBody),
	}))
	// FIXME should we ask for a specific AvailabilityZone here? We don't even have one in the config (it is derived from the network id I think).
	//end := container.FinishedAt.Add(time.Hour * time.Duration(24))
	input := &ec2.DescribeSpotPriceHistoryInput{
		AvailabilityZone: aws.String("us-east-1a"),
		//EndTime: aws.Time(end),
		EndTime: container.FinishedAt,
		//DryRun:  aws.Bool(true),
		InstanceTypes: []*string{
			aws.String(size),
		},
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: container.StartedAt,
	}
	//fmt.Printf("%#v\n", input)

	result, err := svc.DescribeSpotPriceHistory(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return 0, err
	}

	sort.Sort(SpotPriceHistory(result.SpotPriceHistory))

	//fmt.Printf("%#v\n", result)
	var total float64
	last := result.SpotPriceHistory[0]
	last.Timestamp = container.StartedAt
	//fmt.Printf("LAST: %#v\n", last)
	for _, s := range result.SpotPriceHistory[1:] {
		//fmt.Printf("%#v\n", s)
		delta := s.Timestamp.Sub(*last.Timestamp)
		price, err := strconv.ParseFloat(*last.SpotPrice, 64)
		if err != nil {
			return 0, nil
		}
		total += delta.Seconds() / 3600 * price
		last = s
		/*fmt.Printf("YO\n")
		fmt.Printf("%#v\n", s)
		fmt.Printf("COST: %#v\n", price)
		fmt.Printf("TOTAL: %#v\n", delta.Seconds()/3600*price)
		fmt.Printf("SECONDS: %#v\n", delta.Seconds()) */
	}
	delta := container.FinishedAt.Sub(*last.Timestamp)
	price, err := strconv.ParseFloat(*last.SpotPrice, 64)
	if err != nil {
		return 0, nil
	}
	//fmt.Printf("COST: %#v\n", price)
	total += delta.Seconds() / 3600 * price

	return total, err
}
