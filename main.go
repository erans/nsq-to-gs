package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/cihub/seelog"
	"github.com/nsqio/go-nsq"
)

type StringArray []string

func (a *StringArray) Set(s string) error {
	*a = append(*a, s)
	return nil
}

func (a *StringArray) String() string {
	return strings.Join(*a, ",")
}

var (
	showVersion = flag.Bool("version", false, "print version string")

	topic                 = flag.String("topic", "", "NSQ topic")
	channel               = flag.String("channel", "", "NSQ channel")
	maxInFlight           = flag.Int("max-in-flight", 1000, "max number of messages to allow in flight (before flushing)")
	maxInFlightTime       = flag.Int("max-in-flight-time", 60, "max time to keep messages in flight (before flushing)")
	bucketMessages        = flag.Int("bucket-messages", 0, "total number of messages to bucket")
	bucketSeconds         = flag.Int("bucket-seconds", 600, "total time to bucket messages for (seconds)")
	projectID             = flag.String("projectid", "", "Project ID")
	gsBucket              = flag.String("gsbucket", "", "GS bucket-name to store the output on (eg 'nsq-archive'")
	gsPath                = flag.String("gspath", "", "GS path to store files under (eg '/nsq-archive'")
	gsFilePrefix          = flag.String("gsfileprefix", "file", "File name prefix")
	batchMode             = flag.String("batchmode", "memory", "How to batch the messages between flushes [disk, memory, channel]")
	messageBufferFileName = flag.String("bufferfile", "", "Local file to buffer messages in between flushes to GS")
	gsFileExtension       = flag.String("extension", "json", "Extension for files on GS")

	nsqdTCPAddrs     = StringArray{}
	lookupdHTTPAddrs = StringArray{}
)

func init() {
	flag.Var(&nsqdTCPAddrs, "nsqd-tcp-address", "nsqd TCP address (may be given multiple times)")
	flag.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
}

func main() {
	// Make sure we flush the log before quitting:
	defer log.Flush()

	// Process the arguments:
	argumentIssue := processArguments()
	if argumentIssue {
		os.Exit(1)
	}

	// Intercept quit signals:
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Don't ask for more messages than we want
	if *bucketMessages > 0 && *bucketMessages < *maxInFlight {
		*maxInFlight = *bucketMessages
	}

	// Set up the NSQ client:
	cfg := nsq.NewConfig()
	// Had to hardwire the NSQ version as since Go 1.5 one cannot reference
	// a package with the name "internal" in it.
	cfg.UserAgent = fmt.Sprintf("nsq_to_gs/%s go-nsq/%s", "0.3.6", nsq.VERSION)
	cfg.MaxInFlight = *maxInFlight

	consumer, err := nsq.NewConsumer(*topic, *channel, cfg)
	if err != nil {
		panic(err)
	}

	// See which mode we've been asked to run in:
	switch *batchMode {
	case "disk":
		{
			// On-disk:
			messageHandler := &OnDiskHandler{
				allTimeMessages:       0,
				deDuper:               make(map[string]int),
				inFlightMessages:      make([]*nsq.Message, 0),
				timeLastFlushedToGS:   int(time.Now().Unix()),
				timeLastFlushedToDisk: int(time.Now().Unix()),
			}

			// Add the handler:
			consumer.AddHandler(messageHandler)
		}
	case "channel":
		{
			panic("'channel' batch-mode isn't implemented yet!")
		}
	default:
		{
			// Default to in-memory:
			messageHandler := &InMemoryHandler{
				allTimeMessages:     0,
				deDuper:             make(map[string]int),
				messageBuffer:       make([]*nsq.Message, 0),
				timeLastFlushedToGS: int(time.Now().Unix()),
			}

			// Add the handler:
			consumer.AddHandler(messageHandler)
		}
	}

	// Configure the NSQ connection with the list of NSQd addresses:
	err = consumer.ConnectToNSQDs(nsqdTCPAddrs)
	if err != nil {
		panic(err)
	}

	// Configure the NSQ connection with the list of Lookupd HTTP addresses:
	err = consumer.ConnectToNSQLookupds(lookupdHTTPAddrs)
	if err != nil {
		panic(err)
	}

	// Handle stop / quit events:
	for {
		select {
		case <-consumer.StopChan:
			return
		case <-sigChan:
			consumer.Stop()
			os.Exit(0)
		}
	}
}
