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

type config struct {
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
}

type SendArgs struct {
	Content string
}

var logChan chan string

type RpcEndpoint int

var rpcClient *rpc.Client
var wg sync.WaitGroup

func main() {
	config := set_flags()

	logChan = make(chan string, 10000)
	go startLogWriter(logChan, config, &wg)
	startListener(config)
	startSender(config)
	processStdin(logChan, config, &wg)
	wg.Wait()
	//os.Exit(0)
}

func startSender(config *config) {
	if !config.isSending {
		return
	}

	var err error
	rpcClient, err = rpc.DialHTTP("tcp", config.sendHost+":"+strconv.Itoa(config.sendPort))
	if err != nil {
		log.Fatal("dialing:", err)
	}
}

func startListener(config *config) {
	if !config.isListening {
		return
	}

	//wg.Add(1) //not ends
	rpcEndpoint := new(RpcEndpoint)
	rpc.Register(rpcEndpoint)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", config.listenHost+":"+strconv.Itoa(config.listenPort))
	if err != nil {
		log.Fatal("listen error:", err)
	}
	go http.Serve(l, nil)
}

func (t *RpcEndpoint) Send(args *SendArgs, reply *int) error {
	wg.Add(1)
	logChan <- args.Content
	*reply = 0
	return nil
}

func set_flags() *config {
	config := config{}

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

	flag.Parse()
	return &config
}

func startLogWriter(records chan string, config *config, wg *sync.WaitGroup) {
	logFile, err := os.OpenFile(config.logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		panic("Can't open the log file \"" + config.logfile + "\". " + err.Error())
	}

	for {
		logRecord := <-records
		if config.isSending {
			args := &SendArgs{logRecord}
			var reply int
			err := rpcClient.Call("RpcEndpoint.Send", args, &reply)
			if err != nil {
				log.Fatal("Send log record error: ", err)
			}
			wg.Done()
			continue
		}
		_, err := logFile.WriteString(logRecord)
		if err != nil {
			panic("Can't write to the log file. " + err.Error())
		}
		wg.Done()
	}
}

func processStdin(logChan chan string, config *config, wg *sync.WaitGroup) {
	inReader := bufio.NewReader(os.Stdin)
	for {
		line, err := inReader.ReadString('\n')
		if err == io.EOF {
			break
		}

		wg.Add(1)
		logChan <- makeLogString(line, config)
	}
}

func makeLogString(text string, config *config) string {
	logString := text

	if config.nodeName != "" {
		logString = config.nodeName + ": " + logString
	}
	if config.addTimestamp {
		logString = time.Now().UTC().String() + " - " + logString
	}

	return logString
}
