package mms

import "fmt"

type WellKnownParam int

const (
	QParam                WellKnownParam = 0x80
	CharsetParam          WellKnownParam = 0x81
	LevelParam            WellKnownParam = 0x82
	TypeParam             WellKnownParam = 0x83
	DepNameParam          WellKnownParam = 0x85
	DepFilenameParam      WellKnownParam = 0x86
	DifferencesParam      WellKnownParam = 0x87
	PaddingParam          WellKnownParam = 0x88
	CtMrTypeParam         WellKnownParam = 0x89
	DepStartParam         WellKnownParam = 0x8a
	DepStartInfoParam     WellKnownParam = 0x8b
	DepCommentParam       WellKnownParam = 0x8c
	DepDomainParam        WellKnownParam = 0x8d
	MaxAgeParam           WellKnownParam = 0x8e
	DepPathParam          WellKnownParam = 0x8f
	SecureParam           WellKnownParam = 0x90
	SecParam              WellKnownParam = 0x91
	MacParam              WellKnownParam = 0x92
	CreationDateParam     WellKnownParam = 0x93
	ModificationDateParam WellKnownParam = 0x94
	ReadDateParam         WellKnownParam = 0x95
	SizeParam             WellKnownParam = 0x96
	NameParam             WellKnownParam = 0x97
	FilenameParam         WellKnownParam = 0x98
	StartParam            WellKnownParam = 0x99
	StartInfoParam        WellKnownParam = 0x9a
	CommentParam          WellKnownParam = 0x9b
	DomainParam           WellKnownParam = 0x9c
	PathParam             WellKnownParam = 0x9d
)

type PartHeaderField int

const (
	ContentTypePartHeader             PartHeaderField = 0x91
	ContentLocationPartHeader         PartHeaderField = 0x8E
	ContentIDPartHeader               PartHeaderField = 0xC0
	DepContentDispositionPartHeader   PartHeaderField = 0xAE
	ContentDispositionPartHeader      PartHeaderField = 0xC5
	ContentTransferEncodingPartHeader PartHeaderField = 0xc8
)

func (p PartHeaderField) String() string {
	switch p {
	case ContentTypePartHeader:
		return "Content-Type"
	case ContentLocationPartHeader:
		return "Content-Location"
	case ContentIDPartHeader:
		return "Content-ID"
	case DepContentDispositionPartHeader:
		return "Dep-Content-Disposition"
	case ContentDispositionPartHeader:
		return "Content-Disposition"
	case ContentTransferEncodingPartHeader:
		return "Content-Transfer-Encoding"
	default:
		return fmt.Sprintf("UnkownPartHeaderField<%d>", p)
	}
}

type PartDispositionType int

const (
	FormDataDisposition   PartDispositionType = 128
	AttachmentDisposition PartDispositionType = 129
	InlineDisposition     PartDispositionType = 130
)

func (pd PartDispositionType) String() string {
	switch pd {
	case FormDataDisposition:
		return "FormDataDisposition"
	case AttachmentDisposition:
		return "AttachmentDisposition"
	case InlineDisposition:
		return "InlineDisposition"
	default:
		return fmt.Sprintf("UnkownPartDispositionType<%d>", pd)
	}

}
