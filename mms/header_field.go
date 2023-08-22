package mms

import (
	"fmt"
	"strconv"
	"time"
)

type HeaderString string

func (hs *HeaderString) String() string {
	return string(*hs)
}

type HeaderUint uint32

func (hu *HeaderUint) String() string {
	return strconv.Itoa(int(*hu))
}

type HeaderBool bool

func (hb *HeaderBool) String() string {
	return strconv.FormatBool(bool(*hb))
}

type HeaderTime time.Time

func (hd *HeaderTime) String() string {
	return time.Time(*hd).Format(time.RFC3339)
}

type HeaderRelativeOrAbsoluteTime struct {
	Relative *time.Duration
	Absolute *time.Time
}

func (h *HeaderRelativeOrAbsoluteTime) String() string {
	if h.Absolute != nil {
		return h.Absolute.Format(time.RFC3339)
	} else {
		return h.Relative.String()
	}
}

type HeaderMessageType int

const (
	UnknownMessageType HeaderMessageType = 0
	MSendReq           HeaderMessageType = 128
	MSendConf          HeaderMessageType = 129
	MNotificationInd   HeaderMessageType = 130
	MNotifyrespInd     HeaderMessageType = 131
	MRetrieveConf      HeaderMessageType = 132
	MAcknowledgeInd    HeaderMessageType = 133
	MDeliveryInd       HeaderMessageType = 134
)

func (mt *HeaderMessageType) String() string {
	switch *mt {
	case MSendReq:
		return "m-send-req"
	case MSendConf:
		return "m-send-conf"
	case MNotificationInd:
		return "m-notification-ind"
	case MNotifyrespInd:
		return "m-notifyresp-ind"
	case MRetrieveConf:
		return "m-retrieve-conf"
	case MAcknowledgeInd:
		return "m-acknowledge-ind"
	case MDeliveryInd:
		return "m-delivery-ind"
	default:
		return "UnknownMessageType"
	}
}

type HeaderPriority int

const (
	Low    HeaderPriority = 128
	Medium HeaderPriority = 129
	High   HeaderPriority = 130
)

func (p *HeaderPriority) String() string {
	switch *p {
	case Low:
		return "low"
	case Medium:
		return "medium"
	case High:
		return "high"
	default:
		return "UnknownPriority"
	}
}

type HeaderResponseStatus int

const (
	StatusOk                            HeaderResponseStatus = 128
	StatusErrorUnspecified              HeaderResponseStatus = 129
	StatusErrorServiceDenied            HeaderResponseStatus = 130
	StatusErrorMessageFormatCorrupt     HeaderResponseStatus = 131
	StatusErrorSendingAddressUnresolved HeaderResponseStatus = 132
	StatusErrorMessageNotFound          HeaderResponseStatus = 133
	StatusErrorNetworkProblem           HeaderResponseStatus = 134
	StatusErrorContentNotAccepted       HeaderResponseStatus = 135
	StatusErrorUnsupportedMessage       HeaderResponseStatus = 136
)

func (rs *HeaderResponseStatus) String() string {
	switch *rs {
	case StatusOk:
		return "Ok"
	case StatusErrorUnspecified:
		return "Error-unspecified"
	case StatusErrorServiceDenied:
		return "Error-service-denied"
	case StatusErrorMessageFormatCorrupt:
		return "Error-message-format-corrupt"
	case StatusErrorSendingAddressUnresolved:
		return "Error-sending-address-unresolved"
	case StatusErrorMessageNotFound:
		return "Error-message-not-found"
	case StatusErrorNetworkProblem:
		return "Error-network-problem"
	case StatusErrorContentNotAccepted:
		return "Error-content-not-accepted"
	case StatusErrorUnsupportedMessage:
		return "Error-unsupported-message"
	}

	return "Error-unspecified"
}

type HederSenderVisibility int

const (
	Hide HederSenderVisibility = 128
	Show HederSenderVisibility = 129
)

func (v *HederSenderVisibility) String() string {
	switch *v {
	case Hide:
		return "hide"
	case Show:
		return "show"
	}
	return fmt.Sprintf("SenderVisibilityUnknown<%d>", v)
}

type HeaderStatus int

const (
	StatusExpired      HeaderStatus = 128
	StatusRetrieved    HeaderStatus = 129
	StatusRejected     HeaderStatus = 130
	StatusDeferred     HeaderStatus = 131
	StatusUnrecognised HeaderStatus = 132
)

func (s *HeaderStatus) String() string {
	switch *s {
	case StatusExpired:
		return "expired"
	case StatusRetrieved:
		return "retrieved"
	case StatusRejected:
		return "rejected"
	case StatusDeferred:
		return "deferred"
	case StatusUnrecognised:
		return "unrecognised"
	}
	return fmt.Sprintf("StatusUnknown<%d>", s)
}
