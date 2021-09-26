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

func GetAWSSpotPriceHistory(container arvados.Container) (map[string]SpotPriceHistory, error) {
	var history SpotPriceHistory
	// FIXME hardcoded region, does this matter?
	svc := ec2.New(session.New(&aws.Config{
		Region: aws.String("us-east-1"),
		//		LogLevel: aws.LogLevel(aws.LogDebugWithHTTPBody),
	}))
	// FIXME should we ask for a specific AvailabilityZone here? We don't even have one in the config (it is derived from the network id I think).
	input := &ec2.DescribeSpotPriceHistoryInput{
		AvailabilityZone: aws.String("us-east-1a"),
		EndTime:          container.FinishedAt,
		//DryRun:  aws.Bool(true),
		ProductDescriptions: []*string{
			aws.String("Linux/UNIX (Amazon VPC)"),
		},
		StartTime: container.StartedAt,
	}
	//fmt.Printf("%#v\n", input)

	for {
		result, err := svc.DescribeSpotPriceHistory(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				fmt.Println(err.Error())
			}
			return nil, err
		}
		history = append(history, result.SpotPriceHistory...)

		if *result.NextToken == "" {
			// No more pages
			break
		}
		input.NextToken = result.NextToken
	}

	historyMap := make(map[string]SpotPriceHistory)

	for _, sph := range history {
		if _, ok := historyMap[*sph.InstanceType]; !ok {
			historyMap[*sph.InstanceType] = make(SpotPriceHistory, 0)
		}
		historyMap[*sph.InstanceType] = append(historyMap[*sph.InstanceType], sph)
	}

	// Sort all the SpotPriceHistories
	for instanceType := range historyMap {
		sort.Sort(historyMap[instanceType])
	}

	return historyMap, nil
}

func findSpotPriceHistoryStart(container arvados.Container, history SpotPriceHistory) SpotPriceHistory {
	pos := len(history) / 2
	oldPos := pos

	for {
		if history[pos].Timestamp.After(*container.StartedAt) {
			// reduce pos
			oldPos = pos
			pos = pos - pos/2
		}
		if len(history) > pos+1 && history[pos+1].Timestamp.Before(*container.StartedAt) {
			// increase pos
			oldPos = pos
			pos = pos + pos/2
		}

		if oldPos == pos {
			break
		}
		fmt.Printf("pos: %d, oldPos: %d\n", pos, oldPos)
	}

	return history[pos:]
}

func CalculateAWSSpotPrice(container arvados.Container, fullHistory SpotPriceHistory) (float64, error) {
	//fmt.Printf("%#v\n", result)
	history := findSpotPriceHistoryStart(container, fullHistory)
	var total float64
	last := history[0]
	last.Timestamp = container.StartedAt
	//fmt.Printf("LAST: %#v\n", last)
	for _, s := range history[1:] {
		if s.Timestamp.After(*container.FinishedAt) {
			break
		}
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
