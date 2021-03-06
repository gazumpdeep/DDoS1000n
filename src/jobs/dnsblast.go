// MIT License

// Copyright (c) [2022] [Bohdan Ivashko (https://github.com/Arriven)]

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/Arriven/db1000n/src/core/dnsblast"
	"github.com/Arriven/db1000n/src/utils"
)

const (
	defaultProto                   = dnsblast.UDPProtoName
	defaultParallelQueriesPerCycle = 5
	defaultIntervalInMS            = 10
)

type dnsBlastConfig struct {
	BasicJobConfig
	RootDomain      string   `mapstructure:"root_domain"`
	Protocol        string   `mapstructure:"protocol"` // "udp", "tcp", "tcp-tls"
	SeedDomains     []string `mapstructure:"seed_domains"`
	ParallelQueries int      `mapstructure:"parallel_queries"`
}

func dnsBlastJob(ctx context.Context, logger *zap.Logger, globalConfig GlobalConfig, args Args) (data interface{}, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer utils.PanicHandler(logger)

	var jobConfig dnsBlastConfig
	if err = utils.Decode(args, &jobConfig); err != nil {
		return nil, fmt.Errorf("failed to parse DNS Blast job configurations: %w", err)
	}

	//
	// Default settings and early misconfiguration prevention
	//

	// Root domain verification
	if len(jobConfig.RootDomain) == 0 {
		return nil, errors.New("no root domain provided, consider adding it")
	}

	// Domain seeds verification
	if len(jobConfig.SeedDomains) == 0 {
		return nil, errors.New("no seed domains provided, at least one is required")
	}

	// Protocol settlement
	isUDPProto := jobConfig.Protocol == dnsblast.UDPProtoName
	isTCPProto := jobConfig.Protocol == dnsblast.TCPProtoName
	isTCPTLSProto := jobConfig.Protocol == dnsblast.TCPTLSProtoName

	switch {
	case jobConfig.Protocol == "":
		jobConfig.Protocol = defaultProto

	case !(isUDPProto || !isTCPProto || !isTCPTLSProto):
		return nil, fmt.Errorf("unrecognized DNS protocol [provided], expected one of [%v]",
			[]string{dnsblast.UDPProtoName, dnsblast.TCPProtoName, dnsblast.TCPTLSProtoName})
	}

	// Concurrency validation
	if jobConfig.ParallelQueries == 0 {
		jobConfig.ParallelQueries = defaultParallelQueriesPerCycle
	}

	// Delay validation
	if jobConfig.IntervalMs == 0 {
		jobConfig.IntervalMs = defaultIntervalInMS
	}

	var wg sync.WaitGroup

	//
	// Blast the Job!
	//
	err = dnsblast.Start(ctx, logger, &wg, &dnsblast.Config{
		RootDomain:      jobConfig.RootDomain,
		Protocol:        jobConfig.Protocol,
		SeedDomains:     jobConfig.SeedDomains,
		ParallelQueries: jobConfig.ParallelQueries,
		Delay:           time.Duration(jobConfig.IntervalMs) * time.Millisecond,
		ClientID:        globalConfig.ClientID,
	})

	wg.Wait()

	return nil, err
}
