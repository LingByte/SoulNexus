package sip1

import (
	"fmt"
	"net"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/sip1/ua"
	"github.com/emiago/sipgo"
	"github.com/sirupsen/logrus"
)

type SipServer struct {
	config  *ua.UAConfig
	ua      *sipgo.UserAgent
	client  *sipgo.Client
	server  *sipgo.Server
	rtpConn *net.UDPConn
	mutex   sync.RWMutex
	running bool
}

func NewSipServer(rptPort, sipPort int, uaConfig *ua.UAConfig) (*SipServer, error) {
	if uaConfig != nil {
		if err := uaConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid UA config: %w", err)
		}
	} else {
		uaConfig = ua.DefaultUAConfig()
	}
	// 应用默认值
	uaConfig.ApplyDefaults()

	uaConfig.LocalRTPPort = rptPort
	uaConfig.Port = sipPort

	userAgent, err := sipgo.NewUA(sipgo.WithUserAgent(uaConfig.UserAgentName))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create UA")
	}

	server, err := sipgo.NewServer(userAgent)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create SIP server")
	}

	// Create RTP UDP connection
	rtpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", rptPort))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to resolve RTP address")
	}

	rtpConn, err := net.ListenUDP("udp", rtpAddr)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create RTP UDP connection")
	}

	client, err := sipgo.NewClient(userAgent)
	if err != nil {
		logrus.WithError(err).Fatal("Create SIP Client Failed")
	}

	return &SipServer{
		config:  uaConfig,
		server:  server,
		rtpConn: rtpConn,
		client:  client,
		ua:      userAgent,
	}, nil
}

func (as *SipServer) Close() {
	as.server.Close()
	as.rtpConn.Close()
	as.client.Close()
	as.ua.Close()
	as.running = false
	logrus.Info("SIP Server Closed")
}

func (as *SipServer) Start() {
	as.RegisterFunc()

	if err := as.server.ListenAndServe(nil, "udp", fmt.Sprintf("%s:%d", as.config.Host, as.config.Port)); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}
}
