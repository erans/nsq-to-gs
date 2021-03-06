![nsq-to-gs](http://i0.wp.com/eran.sandler.co.il/wp-content/uploads/2015/11/nsq-to-googlestorage.png?resize=300%2C117)

[![Build Status](https://travis-ci.org/erans/nsq-to-gs.svg?branch=master)](https://travis-ci.org/erans/nsq-to-gs)

# nsq-to-gs
Stream an NSQ channel to Google Cloud Storage

Based on [nsq-to-s3](https://github.com/chrusty/nsq-to-s3) by [chrusty](https://github.com/chrusty)

Written (more like adjusted) by Eran Sandler [(@erans)](https://twitter.com/erans) http://eran.sandler.co.il

## Parameters
* _topic:_ The NSQ topic to subscribe to
* _channel:_ An NSQ channel name to use (defaults to an automatically-generated ephemeral channel)
* _max-in-flight:_ The maximum number of unFinished messages to allow (effectively a flush-batch size)
* _max-in-flight-time:_ The maximum number of seconds to wait before flushing (in case maxInFlight is not enough)
* _lookupd-http-address:_ The address of an NSQLookup daemon to connect to
* _nsqd-tcp-address:_ A specific NSQ daemon to connect to
* _bucket-seconds:_ The time-bucket-size of each file you want to end up with on GS, if we don't hit bucketMessages first (eg 3600 will give you one file on GS per-hour)
* _bucket-messages:_ Total number of messages to bucket (if bucketSeconds doesn't elapse first)
* _gsbucket:_ The GS bucket to store the files on (eg "nsq-archive")
* _gspath:_ A path to store the archive files under (eg "/live-dumps")
* _gsfileprefix:_ The generate file name prefix (eg "mylogfile" which would be mylogfile-20151117_1003.json.gz)
* _batchmode:_ Which mode to run in [memory, disk, channel]
* _bufferfile:_ The name of a file to use as a local on-disk buffer between flushes to GS (should be something durable)
* _extension:_ Extension for files on GS (default is json)

## Modes (current)
NSQ-to-GS can operate in several different modes, depending on your storage and/or durability requirements:

### "Batch-on-disk"
  * Subs to NSQ
  * De-dupes in memory (map[string][bool] where string is a hash of the message payload)
  * Once max-in-flight is reached it flushes messages to disk then Finish()es them
  * After timeBucket has elapsed it stops consuming, sticks the file on GS, clears the de-dupe map and continues
  * **You would be well-advised to use some kind of persistent storage**

### "In-memory"
As with batch-on-disk but all messages are kept in-memory between flushes to GS. **If you stop the process then you will lose messages!**

## Modes (planned)

### "Abandoned-channel"
  * Subs to NSQ (creates a channel)
  * Waits for timeBucket to elapse
  * Pauses the channel
  * Takes all the messages off the queue, de-dupes in memory, sticks them on GS
  * Finish()es the messages
  * Unpauses the channel
  * Repeat

### "Continuous-sync-to-gs"
  * As with batch-on-disk but syncs to GS every x seconds
  * Either overwrites the same file on GS, or piles up new ones
  * At the end of the time-bucket the interim files are removed from GS

## Examples

#### Consuming a topic, buffering on disk, flushing in-flight at 1000 messages, flushing to GS every 5 minutes:
```
nsq-to-gs -gsproject=myproject -gsbucket=nsq-archive -topic=firehose -channel='nsq-to-gs#ephemeral' -lookupd-http-address=10.0.0.2:4161 -gspath=/live-dumps/firehose -bucket-seconds=300 -max-in-flight=1000 -batchmode=disk
```

## Bugs (current)
* Dupes can still occur around flush boundaries
* The timer for flushing to GS is based on events arriving (not on absolute time). This means that he filenames/numbers will creep (just being pedantic)
* Should optionally compress files for GS
