package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/trazfr/freebox-exporter/fbx"
	"github.com/trazfr/freebox-exporter/log"
)

func usage(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "[options] <api_token_file>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	debugPtr := flag.Bool("debug", false, "enable the debug mode")
	hostDetailsPtr := flag.Bool("hostDetails", false, "get details about the hosts connected to wifi and ethernet. This increases the number of metrics")
	httpDiscoveryPtr := flag.Bool("httpDiscovery", false, "use http://mafreebox.freebox.fr/api_version to discover the Freebox at the first run (default: mDNS)")
	listenPtr := flag.String("listen", ":9091", "listen to address")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage(errors.New("api_token_file not defined"))
	} else if len(args) > 1 {
		usage(errors.New("too many arguments"))
	}
	if *debugPtr {
		log.Init(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	} else {
		log.Init(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	}
	discovery := fbx.FreeboxDiscoveryMDNS
	if *httpDiscoveryPtr {
		discovery = fbx.FreeboxDiscoveryHTTP
	}

	collector := NewCollector(args[0], discovery, *hostDetailsPtr, *debugPtr)
	defer collector.Close()

	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	log.Error.Println(http.ListenAndServe(*listenPtr, nil))
}
