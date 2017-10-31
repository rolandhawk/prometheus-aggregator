package server

import (
	"net"
	"syscall"
	"time"

	pr "github.com/bukalapak/prometheus-aggregator"
	"github.com/bukalapak/prometheus-aggregator/protomodel"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/log"
)

type Server struct {
	Protor *pr.Protor
	buf    []byte
}

func New(pProtor *pr.Protor) *Server {
	s := Server{
		Protor: pProtor,
		buf:    make([]byte, 1024),
	}
	return &s
}

func MetricProtor(Name string, Kind string, Value float64, start time.Time) *pr.Sample {
	HD := []float64{}
	if Kind == "h" {
		Value = float64(time.Since(start).Nanoseconds())
		HD = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	}
	return &pr.Sample{
		Service:      "metrics",
		Name:         Name,
		Kind:         Kind,
		Label:        make(map[string]string),
		Value:        Value,
		HistogramDef: HD,
	}
}

func (s *Server) Run(ip string, port string) {
	laddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		exitOnFatal(err, "Can't resolve address")
	}
	tcp, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		exitOnFatal(err, "Fail at starting server")
	}
	for {
		conn, err := tcp.Accept()
		if err != nil {
			exitOnFatal(err, "Fail at connecting to incoming connection")
		}
		go s.handleRequest(conn)
	}
}

func (s *Server) handleRequest(conn net.Conn) {
	sTime := time.Now()
	reqLen, err := conn.Read(s.buf)
	if err != nil {
		log.Info("err at reading ", err)
		return
	}
	protodata := &protomodel.Array{}
	err = proto.Unmarshal(s.buf[0:reqLen], protodata)
	if err != nil {
		log.Info("err at unmarshal protobuff ", err)
		return
	}
	for _, sample := range protodata.Samples {
		err = s.Protor.WriteToRegistry(ProtoToSample(sample))
		if err != nil {
			log.Info("error at writing to registry", err)
		}
	}

	err = s.Protor.WriteToRegistry(MetricProtor("app_request_handle_duration_ns", "h", 1, sTime))
	err = s.Protor.WriteToRegistry(MetricProtor("app_total_sample", "c", float64(len(protodata.Samples)), sTime))

}

func ProtoToSample(pd *protomodel.Array_Sample) *pr.Sample {
	return &pr.Sample{
		Service:      pd.Service,
		Name:         pd.Name,
		Kind:         pd.Kind,
		Label:        pd.Label,
		Value:        pd.Value,
		HistogramDef: pd.HistogramDef,
	}
}

func exitOnFatal(err error, loc string) {
	log.Fatalf("EXIT on %s: err=%s\n", loc, err)
	syscall.Exit(1)
}