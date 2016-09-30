package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

type simplogConfig struct {
	logfile      string
	isListening  bool
	listenHost   string
	listenPort   int
	isSending    bool
	sendHost     string
	sendPort     int
	addTimestamp bool
	nodeName     string
	period       time.Duration
	debug        bool
}

type SendArgs struct {
	Content string
}

var logChan chan string

type RpcEndpoint int

var rpcClient *rpc.Client
var wg sync.WaitGroup
var config simplogConfig

func main() {
	set_flags()
	if config.debug {
		log.Printf("Config: %#v\n", config)
	}

	logChan = make(chan string, 10000)
	go startLogWriter(logChan, &wg)
	startListener()
	startSender()
	processStdin(logChan, &wg)
	if config.debug {
		log.Print("Blocked on group waiting\n")
	}
	wg.Wait()
}

func startSender() {
	if !config.isSending {
		return
	}

	if config.debug {
		log.Print("Starting RPC client\n")
	}
	var err error
	rpcClient, err = rpc.DialHTTP("tcp", config.sendHost+":"+strconv.Itoa(config.sendPort))
	if err != nil {
		log.Fatal("dialing:", err)
	}
}

func startListener() {
	if !config.isListening {
		return
	}

	if config.debug {
		log.Print("Starting RPC server\n")
	}
	rpcEndpoint := new(RpcEndpoint)
	rpc.Register(rpcEndpoint)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", config.listenHost+":"+strconv.Itoa(config.listenPort))
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go http.Serve(l, nil)
	wg.Add(1)
}

func (t *RpcEndpoint) Send(args *SendArgs, reply *int) error {
	if config.debug {
		log.Printf("Received new RPC Send call with args: %#v\n Sending new content to the log writer\n", args)
	}
	wg.Add(1)
	logChan <- makeLogString(args.Content)
	*reply = 0
	return nil
}

func set_flags() {
	flag.BoolVar(&config.isListening, "listen", false, "Description")
	flag.StringVar(&config.listenHost, "listen-host", "localhost", "Description")
	flag.IntVar(&config.listenPort, "listen-port", 22016, "Description")
	flag.BoolVar(&config.isSending, "send", false, "Description")
	flag.StringVar(&config.sendHost, "send-host", "localhost", "Description")
	flag.IntVar(&config.sendPort, "send-port", 22016, "Description")
	flag.BoolVar(&config.addTimestamp, "timestamp", true, "Description")
	flag.StringVar(&config.nodeName, "name", "", "Description")
	flag.StringVar(&config.logfile, "logfile", "./simplog.log", "Description")
	flag.DurationVar(&config.period, "period", time.Duration(24*365*42)*time.Hour, "Description 1d 1w")
	flag.BoolVar(&config.debug, "debug", false, "Set the flag to enable internal logging")

	flag.Parse()
}

func startLogWriter(records chan string, wg *sync.WaitGroup) {
	if config.debug {
		log.Printf("Open logfile: %#v\n", config.logfile)
	}
	logFile, err := os.OpenFile(config.logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		panic("Can't open the log file \"" + config.logfile + "\". " + err.Error())
	}

	for {
		if config.debug {
			log.Print("Wait for new content for writing to logfile\n")
		}
		logRecord := <-records
		if config.debug {
			log.Printf("Received new content for writing to logfile: %#v\n", logRecord)
		}
		if config.isSending {
			args := &SendArgs{logRecord}
			var reply int
			if config.debug {
				log.Printf("Start RPC call with args: %#v\n", args)
			}
			err := rpcClient.Call("RpcEndpoint.Send", args, &reply)
			if err != nil {
				log.Fatal("Send log record error: ", err)
			}
			if config.debug {
				log.Print("RPC call done successfuly\n", args)
			}
			wg.Done()
			continue
		}
		if config.debug {
			log.Print("Write new content to the logFile\n")
		}
		_, err := logFile.WriteString(logRecord)
		if err != nil {
			panic("Can't write to the log file. " + err.Error())
		}
		wg.Done()
	}
}

func processStdin(logChan chan string, wg *sync.WaitGroup) {
	if config.debug {
		log.Print("Start processing stdin\n")
	}
	inReader := bufio.NewReader(os.Stdin)
	for {
		line, err := inReader.ReadString('\n')
		if config.debug {
			log.Printf("Received new line from stdin: %#v", line)
		}
		if err == io.EOF {
			if config.debug {
				log.Print("stdin EOF\n")
			}
			break
		}

		if config.debug {
			log.Print("Sending the new line to the log writer\n")
		}
		wg.Add(1)
		logChan <- makeLogString(line)
	}
}

func makeLogString(text string) string {
	logString := text

	if config.nodeName != "" {
		logString = config.nodeName + ": " + logString
	}
	if config.addTimestamp && !config.isSending {
		logString = time.Now().UTC().String() + " - " + logString
	}
	if config.debug {
		log.Printf("Making the log string: from %#v to %#v \n", text, logString)
	}

	return logString
}
