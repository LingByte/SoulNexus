package system

import "sync/atomic"

// OpsCounters tracks process-wide error / anomaly counters for dashboards.
type OpsCounters struct {
	PanicTotal           uint64 `json:"panicTotal"`
	HTTP5xxTotal         uint64 `json:"http5xxTotal"`
	HTTP4xxTotal         uint64 `json:"http4xxTotal"`
	JWTVerifyFailTotal   uint64 `json:"jwtVerifyFailTotal"`
	DBReconnectTotal     uint64 `json:"dbReconnectTotal"`
	PreflightFailTotal   uint64 `json:"preflightFailTotal"`
	DependencyFailTotal  uint64 `json:"dependencyFailTotal"`
	FileIOErrorTotal     uint64 `json:"fileIoErrorTotal"`
	SSLCertLoadFailTotal uint64 `json:"sslCertLoadFailTotal"`
}

var (
	panicTotal           atomic.Uint64
	http5xxTotal         atomic.Uint64
	http4xxTotal         atomic.Uint64
	jwtVerifyFailTotal   atomic.Uint64
	dbReconnectTotal     atomic.Uint64
	preflightFailTotal   atomic.Uint64
	dependencyFailTotal  atomic.Uint64
	fileIOErrorTotal     atomic.Uint64
	sslCertLoadFailTotal atomic.Uint64
)

func GetOpsCounters() OpsCounters {
	return OpsCounters{
		PanicTotal:           panicTotal.Load(),
		HTTP5xxTotal:         http5xxTotal.Load(),
		HTTP4xxTotal:         http4xxTotal.Load(),
		JWTVerifyFailTotal:   jwtVerifyFailTotal.Load(),
		DBReconnectTotal:     dbReconnectTotal.Load(),
		PreflightFailTotal:   preflightFailTotal.Load(),
		DependencyFailTotal:  dependencyFailTotal.Load(),
		FileIOErrorTotal:     fileIOErrorTotal.Load(),
		SSLCertLoadFailTotal: sslCertLoadFailTotal.Load(),
	}
}

func IncPanic()           { panicTotal.Add(1) }
func IncHTTP5xx()         { http5xxTotal.Add(1) }
func IncHTTP4xx()         { http4xxTotal.Add(1) }
func IncJWTVerifyFail()   { jwtVerifyFailTotal.Add(1) }
func IncDBReconnect()     { dbReconnectTotal.Add(1) }
func IncPreflightFail()   { preflightFailTotal.Add(1) }
func IncDependencyFail()  { dependencyFailTotal.Add(1) }
func IncFileIOError()     { fileIOErrorTotal.Add(1) }
func IncSSLCertLoadFail() { sslCertLoadFailTotal.Add(1) }
