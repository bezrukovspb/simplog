package main

import (
	"flag"
	"fmt"
)

type config struct {
	path          string
	isListening   bool
	listen_host   string
	listen_port   int
	isSending     bool
	send_host     string
	send_port     int
	add_timestamp bool
	node_name     string
	period        string
}

func main() {
	config := set_flags()
	fmt.Println(config.isListening, config.isSending, config.listen_port, config.listen_host, config.send_host, config.send_port, config.add_timestamp, config.node_name)
}

func set_flags() *config {
	config := config{}

	flag.BoolVar(&config.isListening, "listen", false, "Description")
	flag.StringVar(&config.listen_host, "listen-host", "localhost", "Description")
	flag.IntVar(&config.listen_port, "listen-port", 22016, "Description")
	flag.BoolVar(&config.isSending, "send", false, "Description")
	flag.StringVar(&config.send_host, "send-host", "localhost", "Description")
	flag.IntVar(&config.send_port, "send-port", 22016, "Description")
	flag.BoolVar(&config.add_timestamp, "timestamp", true, "Description")
	flag.StringVar(&config.node_name, "name", "", "Description")
	flag.StringVar(&config.path, "path", "./simplog.log", "Description")
	flag.StringVar(&config.period, "period", "none", "Description 1d 1w")

	flag.Parse()
	return &config
}
