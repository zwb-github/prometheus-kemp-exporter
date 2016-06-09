package cmd

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	kemp "github.com/giantswarm/kemp-client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server [endpoint] [username] [password]",
		Short: "Start the HTTP server",
		Run:   serverRun,
	}

	debug       bool
	waitSeconds int
	port        int

	connsPerSec = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kemp_totals_connections_per_second",
		Help: "The number of connections per second.",
	})
	bytesPerSec = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kemp_totals_bytes_per_second",
		Help: "The number of bytes per second.",
	})
	packetsPerSec = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kemp_totals_packets_per_second",
		Help: "The number of packets per second.",
	})

	virtualServerTotalConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_total_connections",
		Help: "The number of total connections per virtual server.",
	}, []string{"address", "port"})
	virtualServerTotalPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_total_packets",
		Help: "The number of total packets per virtual server.",
	}, []string{"address", "port"})
	virtualServerTotalBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_total_bytes",
		Help: "The number of total bytes per virtual server.",
	}, []string{"address", "port"})
	virtualServerActiveConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_active_connections",
		Help: "The number of active connections per virtual server.",
	}, []string{"address", "port"})
	virtualServerConnsPerSec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_connections_per_second",
		Help: "The number of connections per second per virtual server.",
	}, []string{"address", "port"})
	virtualServerBytesRead = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_bytes_read",
		Help: "The number of bytes read per virtual server.",
	}, []string{"address", "port"})
	virtualServerBytesWritten = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_virtual_server_bytes_written",
		Help: "The number of bytes written per virtual server",
	}, []string{"address", "port"})

	realServerTotalConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_total_connections",
		Help: "The number of total connections per real server.",
	}, []string{"address", "port"})
	realServerTotalPackets = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_total_packets",
		Help: "The number of total packets per real server.",
	}, []string{"address", "port"})
	realServerTotalBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_total_bytes",
		Help: "The number of total bytes per real server.",
	}, []string{"address", "port"})
	realServerActiveConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_active_connections",
		Help: "The number of active connections per real server.",
	}, []string{"address", "port"})
	realServerConnsPerSec = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_connections_per_second",
		Help: "The number of connections per second per real server.",
	}, []string{"address", "port"})
	realServerBytesRead = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_bytes_read",
		Help: "The number of bytes read per real server.",
	}, []string{"address", "port"})
	realServerBytesWritten = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kemp_real_server_bytes_written",
		Help: "The number of bytes written per real server",
	}, []string{"address", "port"})
)

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVar(&port, "port", 8000, "port to listen on")
	serverCmd.Flags().IntVar(&waitSeconds, "wait", 10, "time (in seconds) between accessing the Kemp API")
	serverCmd.Flags().BoolVar(&debug, "debug", false, "enable debug output")

	prometheus.MustRegister(connsPerSec)
	prometheus.MustRegister(bytesPerSec)
	prometheus.MustRegister(packetsPerSec)

	prometheus.MustRegister(virtualServerTotalConnections)
	prometheus.MustRegister(virtualServerTotalPackets)
	prometheus.MustRegister(virtualServerTotalBytes)
	prometheus.MustRegister(virtualServerActiveConnections)
	prometheus.MustRegister(virtualServerConnsPerSec)
	prometheus.MustRegister(virtualServerBytesRead)
	prometheus.MustRegister(virtualServerBytesWritten)

	prometheus.MustRegister(realServerTotalConnections)
	prometheus.MustRegister(realServerTotalPackets)
	prometheus.MustRegister(realServerTotalBytes)
	prometheus.MustRegister(realServerActiveConnections)
	prometheus.MustRegister(realServerConnsPerSec)
	prometheus.MustRegister(realServerBytesRead)
	prometheus.MustRegister(realServerBytesWritten)
}

func serverRun(cmd *cobra.Command, args []string) {
	flag.Parse()

	if len(cmd.Flags().Args()) != 3 {
		cmd.Help()
		os.Exit(1)
	}

	client := kemp.NewClient(kemp.Config{
		Endpoint: flag.Arg(1),
		User:     flag.Arg(2),
		Password: flag.Arg(3),
		Debug:    debug,
	})

	go func() {
		for {
			statistics, err := client.GetStatistics()
			if err != nil {
				log.Println("Error getting statistics ", err)
				os.Exit(1)
			}

			connsPerSec.Set(float64(statistics.Totals.ConnectionsPerSec))
			bytesPerSec.Set(float64(statistics.Totals.BytesPerSec))
			packetsPerSec.Set(float64(statistics.Totals.PacketsPerSec))

			for _, vs := range statistics.VirtualServers {
				virtualServerTotalConnections.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.TotalConnections))
				virtualServerTotalPackets.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.TotalPackets))
				virtualServerTotalBytes.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.TotalBytes))
				virtualServerActiveConnections.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.ActiveConnections))
				virtualServerConnsPerSec.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.ConnectionsPerSec))
				virtualServerBytesRead.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.BytesRead))
				virtualServerBytesWritten.WithLabelValues(vs.Address, strconv.Itoa(vs.Port)).Set(float64(vs.BytesWritten))
			}

			for _, rs := range statistics.RealServers {
				realServerTotalConnections.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.TotalConnections))
				realServerTotalPackets.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.TotalPackets))
				realServerTotalBytes.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.TotalBytes))
				realServerActiveConnections.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.ActiveConnections))
				realServerConnsPerSec.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.ConnectionsPerSec))
				realServerBytesRead.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.BytesRead))
				realServerBytesWritten.WithLabelValues(rs.Address, strconv.Itoa(rs.Port)).Set(float64(rs.BytesWritten))
			}

			time.Sleep(time.Second * time.Duration(waitSeconds))
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "")
	})
	http.Handle("/metrics", prometheus.Handler())

	log.Print("Listening on port ", port)

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
