package mms

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"time"
)

type Message struct {
	Header map[MMSField][]HeaderField
	Parts  []PDUPart
}

type HeaderField interface {
	String() string
}

func Unmarshal(packet []byte) (*Message, error) {
	rr := bytes.NewReader(packet)

	dec := decoder{
		r:      bufio.NewReader(rr),
		seeker: rr,
	}

	hdr, err := dec.decodeHeader()
	if err != nil {
		return nil, err
	}

	parts, err := dec.decodeBody()
	if err != nil && err != io.EOF {
		return nil, err
	}

	msg := Message{
		Header: hdr,
		Parts:  parts,
	}

	return &msg, nil
}

type decoder struct {
	r      *bufio.Reader
	seeker io.Seeker
	err    error
}

type PDUPart struct {
	Header      map[string]string
	FileName    string
	ContentType string
	Data        []byte
}

func (d *decoder) decodeBody() ([]PDUPart, error) {
	entries, err := d.decodeVarUint()
	if err != nil {
		return nil, err
	}

	var parts []PDUPart

	for i := 0; i < int(entries); i++ {
		part := PDUPart{
			Header: make(map[string]string),
		}
		headerLen, err := d.decodeVarUint()
		if err != nil {
			return nil, err
		}
		dataLen, err := d.decodeVarUint()
		if err != nil {
			return nil, err
		}

		headerBuf := make([]byte, headerLen)
		n, err := io.ReadFull(d.r, headerBuf)
		if err != nil {
			return nil, fmt.Errorf("read mime part header err: %w, n:%d want:%d", err, n, headerLen)
		}
		rr := bytes.NewReader(headerBuf)
		tmpDecoder := decoder{
			r:      bufio.NewReader(rr),
			seeker: rr,
		}

		s, params, err := tmpDecoder.decodeContentTypeValue()
		if err != nil {
			return nil, fmt.Errorf("decode content type for mime part err: %w", err)
		}

		part.ContentType = s
		if typ, ok := params[TypeParam]; ok {
			part.Header["Content-Type"] = typ
		}
		for k, v := range params {
			switch k {
			case TypeParam:
				part.Header["Content-Type"] = v
			case NameParam:
				part.Header["Name"] = v
			case CharsetParam:
				part.Header["Character-Set"] = v
			case StartParam:
				part.Header["Start"] = v
			}
		}

		if int(tmpDecoder.offset()) < len(headerBuf) {
			return nil, fmt.Errorf("pending part headers")
		}

		filename, headers, err := tmpDecoder.decodePartHeaders()
		if err != nil {
			return nil, fmt.Errorf("parse mime part header err: %w", err)
		}

		part.FileName = filename
		for k, v := range headers {
			part.Header[k] = v
		}

		body := make([]byte, dataLen)
		_, err = io.ReadFull(d.r, body)
		if err != nil {
			return nil, fmt.Errorf("read mime part body err %w", err)
		}

		part.Data = body

		parts = append(parts, part)
	}

	return parts, nil
}

// WAP-209: section 7.1
//
//	Header = MMS-header | Application-header
func (d *decoder) decodeHeader() (map[MMSField][]HeaderField, error) {
	if d.err != nil {
		return nil, d.err
	}

	hdr := make(map[MMSField][]HeaderField)

OUTER:
	for {
		mmsFieldType, err := d.decodeFieldType()
		if err == io.EOF {
			break
		} else if err != nil {
			d.err = err
			return nil, err
		}

		if mmsFieldType == 0 {
			break
		}

		switch mmsFieldType {
		case Bcc, Cc, ResponseText, Subject, To:
			str, err := d.decodeEncodedString()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(str)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)
		case From:
			from, err := d.decodeFrom()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(from)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)
		case DeliveryReport, ReadReply, ReportAllowed:
			val, err := d.decodeBoolean()
			if err != nil {
				d.err = err
				return nil, err
			}
			hb := HeaderBool(val)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hb)
		case ContentType:
			val, _, err := d.decodeContentTypeValue()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(val)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)

			// ContentType will be the last header
			break OUTER
		case Date:
			date, err := d.decodeDate()
			if err != nil {
				d.err = err
				return nil, err
			}
			hd := HeaderTime(date)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hd)
		case DeliveryTime, Expiry:
			dt, err := d.decodeRelativeOrAbsoluteTime()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], dt)
		case MessageSize:
			size, err := d.decodeLongInt()
			if err != nil {
				d.err = err
				return nil, err
			}
			hu := HeaderUint(size)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hu)
		case MessageClass:
			cls, err := d.decodeMessageClass()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(cls)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)
		case MessageID, ContentLocation, TransactionID:
			txt, err := d.decodeTextEnc()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(txt)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)
		case MessageType:
			typ, err := d.decodeMessageType()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &typ)
		case MMSVersion:
			version, err := d.decodeVersion()
			if err != nil {
				d.err = err
				return nil, err
			}
			hs := HeaderString(version)
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &hs)
		case Priority:
			priority, err := d.decodePriority()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &priority)
		case ResponseStatus:
			status, err := d.decodeResponseStatus()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &status)
		case SenderVisibility:
			vis, err := d.decodeSenderVisibility()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &vis)
		case StatusField:
			status, err := d.decodeStatus()
			if err != nil {
				d.err = err
				return nil, err
			}
			hdr[mmsFieldType] = append(hdr[mmsFieldType], &status)

		case RetrieveStatus:
			// XXXX don't populate
			d.r.ReadByte()

		default:
			d.err = fmt.Errorf("unknown mms field type %s", mmsFieldType)
			return nil, d.err
		}
	}

	return hdr, nil
}

// decode a message multipart headers
func (d *decoder) decodePartHeaders() (string, map[string]string, error) {
	resp := make(map[string]string)
	var fileName string
	for {

		peekBuf, err := d.r.Peek(1)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, err
		}
		b := peekBuf[0]

		if b > 127 {
			// numeric assigned header
			header := PartHeaderField(b)
			switch header {
			case ContentLocationPartHeader, ContentIDPartHeader:
				txt, err := d.decodeTextEnc()
				if err != nil {
					return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
				}

				resp[header.String()] = txt
			case ContentDispositionPartHeader, DepContentDispositionPartHeader:
				// Content-disposition-value = Value-length Disposition *(Parameter)
				// Disposition = Form-data | Attachment | Inline | Token-text
				// Form-data = <Octet 128>
				// Attachment = <Octet 129>
				// Inline = <Octet 130>

				len, err := d.decodeValueLength()
				if err != nil {
					return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
				}

				buf := make([]byte, int(len))
				_, err = io.ReadFull(d.r, buf)
				if err != nil {
					return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
				}

				rr := bytes.NewReader(buf)
				tmpDecoder := decoder{
					r:      bufio.NewReader(rr),
					seeker: rr,
				}

				peekBuf, err = tmpDecoder.r.Peek(1)
				if err != nil {
					return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
				}

				b := peekBuf[0]
				if b > 127 {
					tmpDecoder.r.ReadByte()
					resp[header.String()] = PartDispositionType(b).String()
				} else {
					txt, err := tmpDecoder.decodeTextEnc()
					if err != nil {
						return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
					}

					resp[header.String()] = txt
				}

				params, err := tmpDecoder.decodeContentTypeParams()
				if err != nil {
					return "", nil, fmt.Errorf("parse %s header part err: %w", header, err)
				}

				fileName = params[FilenameParam]

			default:
				return "", nil, fmt.Errorf("parse %s header part err: unknown header", header)
			}
		} else {
			name, err := d.decodeTextEnc()
			if err != nil {
				return "", nil, err
			}
			val, err := d.decodeTextEnc()
			if err != nil {
				return "", nil, err
			}

			resp[name] = val
		}

	}

	return fileName, resp, nil
}

func (d *decoder) decodeEncodedString() (string, error) {
	// 7.2.9. Encoded-string-value
	// Encoded-string-value = Text-string | Value-length Char-set Text-string
	// The Char-set values are registered by IANA as MIBEnum value.

	peakbuf, err := d.r.Peek(1)
	if err != nil {
		return "", err
	}
	b := peakbuf[0]

	if b < 32 {
		// Value-length first byte is b < 32
		l, err := d.decodeValueLength()
		if err != nil {
			return "", err
		}
		buf := make([]byte, int(l))
		_, err = io.ReadFull(d.r, buf)
		if err != nil {
			return "", err
		}

		if len(buf) < 1 {
			return "", fmt.Errorf("invalid empty encoded string")
		}

		if buf[0] == 127 { // the any charset
			return string(buf[1:]), nil
		}

		rr := bytes.NewReader(buf)
		tmpDecoder := decoder{
			r:      bufio.NewReader(rr),
			seeker: rr,
		}

		if buf[0] > 32 {
			return tmpDecoder.decodeTextEnc()
		}

		contentTypeIdx, err := tmpDecoder.decodeLongInt()
		if err != nil {
			return "", err
		}

		// TODO(psanford): need to handle the different content types here
		_ = contentTypeIdx

		text, err := io.ReadAll(rr)
		if err != nil {
			return "", err
		}
		if len(text) <= 1 {
			return "", nil
		}
		return string(text[:len(text)-1]), nil

	} else {
		text, err := d.r.ReadBytes(0)
		if err != nil {
			return "", err
		}
		if len(text) <= 1 {
			return "", nil
		}
		return string(text[len(text)-1]), nil
	}

}

func (d *decoder) offset() int64 {
	i, _ := d.seeker.Seek(0, io.SeekCurrent)
	return i
}

func (d *decoder) decodeFieldType() (MMSField, error) {
	peekBytes, err := d.r.Peek(1)
	if err != nil {
		return 0, err
	}
	b := peekBytes[0]
	if b&0x80 != 0x80 {
		return 0, fmt.Errorf("invalid short int at pos:%d, value: 0x%x", d.offset()-1, b)
	}
	f := b & 0x7f
	d.r.ReadByte()
	return MMSField(f), nil
}

func (d *decoder) decodeBoolean() (bool, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return false, err
	}

	switch b {
	case 128:
		return true, nil
	case 129:
		return false, nil
	}

	return false, fmt.Errorf("Invalid boolean value at pos:%d, value: 0x%x", d.offset()-1, b)
}

func (d *decoder) decodeLongInt() (uint32, error) {
	// 	Long-integer = Short-length Multi-octet-integer
	// ; The Short-length indicates the length of the Multi-octet-integer
	// Multi-octet-integer = 1*30 OCTET
	// ; The content octets shall be an unsigned integer value
	// ; with the most significant octet encoded first (big-endian representation
	shortLen, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if shortLen > 30 {
		return 0, fmt.Errorf("invalid long int at pos:%d, shortLen: 0x%x", d.offset()-1, shortLen)
	}

	if shortLen > 8 {
		return 0, fmt.Errorf("unsupported long int at pos:%d, byte size: %d", d.offset()-1, shortLen)
	}

	var u uint32
	for i := 0; i < int(shortLen); i++ {
		b, err := d.r.ReadByte()
		if err != nil {
			return 0, err
		}
		u <<= 8
		u |= uint32(b)
	}

	return u, nil
}

func (d *decoder) decodeShortInt() (byte, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b&0x80 != 0x80 {
		return 0, fmt.Errorf("invalid short int at pos:%d, value: 0x%x", d.offset()-1, b)
	}
	return b & 0x7f, nil
}

func (d *decoder) decodeMessageClass() (string, error) {
	// 7.2.12. Message-Class field
	// Message-class-value = Class-identifier | Token-text
	// Class-identifier = Personal | Advertisement | Informational | Auto
	// Personal = <Octet 128>
	// Advertisement = <Octet 129>
	// Informational = <Octet 130>
	// Auto = <Octet 131>
	// The token-text is an extension method to the message class.

	peakbuf, err := d.r.Peek(1)
	if err != nil {
		return "", err
	}
	b := peakbuf[0]

	if b < 127 {
		text, err := d.r.ReadBytes(0)
		if err != nil {
			return "", err
		}
		if len(text) < 2 {
			return "", nil
		}

		return string(text[:len(text)-1]), nil
	}

	d.r.ReadByte()

	switch b {
	case 128:
		return "personal", nil
	case 129:
		return "advertisement", nil
	case 130:
		return "informational", nil
	case 131:
		return "auto", nil
	default:
		return fmt.Sprintf("UnknownMessageClass<%d>", b), nil
	}
}

func (d *decoder) decodeDate() (time.Time, error) {
	i, err := d.decodeLongInt()
	if err != nil {
		return time.Time{}, err
	}

	t := time.Unix(int64(i), 0)
	return t, nil
}

func (d *decoder) decodeRelativeOrAbsoluteTime() (*HeaderRelativeOrAbsoluteTime, error) {
	// 	7.2.7. Delivery-Time field
	// Delivery-time-value = Value-length (Absolute-token Date-value | Relative-token Delta-seconds-value)
	// Absolute-token = <Octet 128>
	// Relative-token = <Octet 129>

	d.decodeValueLength()

	const (
		absolute = 128
		relative = 129
	)

	mode, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}
	val, err := d.decodeLongInt()
	if err != nil {
		return nil, err
	}

	var result HeaderRelativeOrAbsoluteTime

	switch mode {
	case absolute:
		ts := time.Unix(int64(val), 0)
		result.Absolute = &ts
	case relative:
		d := time.Duration(int64(val)) * time.Second
		result.Relative = &d
	default:
		return nil, fmt.Errorf("invalid delivery_time mode: 0x%x", mode)
	}

	return &result, nil
}

func (d *decoder) decodeContentTypeValue() (string, map[WellKnownParam]string, error) {
	// 8.4.2.7 Accept field
	// The following rules are used to encode accept values.
	// Accept-value = Constrained-media | Accept-general-form
	// Accept-general-form = Value-length Media-range [Accept-parameters]
	// Media-range = (Well-known-media | Extension-Media) *(Parameter)
	// Accept-parameters = Q-token Q-value *(Accept-extension)
	// Accept-extension = Parameter
	// Constrained-media = Constrained-encoding
	// Well-known-media = Integer-value
	// ; Both are encoded using values from Content Type Assignments table in Assigned Numbers
	// Q-token = <Octet 128>

	// constrained-media = Constrained-encoding
	// Constrained-encoding = Extension-Media | Short-integer
	// Short-integer = u8 > 127

	peakbuf, err := d.r.Peek(1)
	if err != nil {
		return "", nil, err
	}
	b := peakbuf[0]

	if b < 32 {
		// Accept-general-form = Value-length Media-range
		// Value-length first byte is b < 32
		l, err := d.decodeValueLength()
		if err != nil {
			return "", nil, err
		}
		buf := make([]byte, int(l))
		_, err = io.ReadFull(d.r, buf)
		if err != nil {
			return "", nil, err
		}

		rr := bytes.NewReader(buf)
		tmpDecoder := decoder{
			r:      bufio.NewReader(rr),
			seeker: rr,
		}
		contentType, err := tmpDecoder.decodeConstrainedMedia()
		if err != nil {
			return "", nil, err
		}

		params, err := tmpDecoder.decodeContentTypeParams()
		if err != nil {
			return "", nil, fmt.Errorf("decode content type params err: %w", err)
		}

		return contentType, params, nil

	} else {
		// Constrained-media = Constrained-encoding
		contentType, err := d.decodeConstrainedMedia()
		return contentType, nil, err
	}
}

func (d *decoder) decodeContentTypeParams() (map[WellKnownParam]string, error) {
	out := make(map[WellKnownParam]string)
	for {
		b, err := d.r.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			d.err = err
			return nil, err
		}

		param := WellKnownParam(b)
		switch param {
		case TypeParam, CtMrTypeParam:
			// type = Constrained-encoding
			// Constrained-encoding = Extension-Media | Short-integer
			// Extension-media = *TEXT End-of-string

			peakbuf, err := d.r.Peek(1)
			if err != nil {
				return nil, err
			}
			b := peakbuf[0]

			if b > 127 {
				idx, err := d.decodeShortInt()
				if err != nil {
					return nil, err
				}
				if int(idx) < len(contentTypes) {
					contentType := contentTypes[int(idx)]
					out[TypeParam] = contentType
				}
			} else {
				text, err := d.decodeTextEnc()
				if err != nil {
					return nil, err
				}
				out[TypeParam] = text
			}
		case StartParam, DepStartParam:
			text, err := d.decodeTextEnc()
			if err != nil {
				return nil, err
			}
			out[StartParam] = text
		case CharsetParam:
			peakbuf, err := d.r.Peek(1)
			if err != nil {
				return nil, err
			}
			b := peakbuf[0]
			if b < 127 {
				// Extension-Media = *TEXT End-of-string
				// *TEXT = byte array where each byte > 31 < 127
				text, err := d.r.ReadBytes(0)
				if err != nil {
					return nil, err
				}
				if len(text) <= 1 {
					continue
				}
				out[CharsetParam] = string(text[len(text)-1])
			} else {
				b, err = d.decodeShortInt()
				if err != nil {
					return nil, err
				}
				if int(b) < len(contentTypes) {
					out[CharsetParam] = contentTypes[b]
				}
			}

		case NameParam, DepNameParam:
			name, err := d.decodeTextEnc()
			if err != nil {
				return nil, err
			}
			out[NameParam] = name
		}
	}

	return out, nil
}

func (d *decoder) decodeValueLength() (uint32, error) {
	// 8.4.2.2 Length
	// The following rules are used to encode length indicators.
	// Value-length = Short-length | (Length-quote Length)
	// ; Value length is used to indicate the length of the value to follow
	// Short-length = <Any octet 0-30>
	// Length-quote = <Octet 31>
	// Length = Uintvar-integer
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b < 31 {
		return uint32(b), nil
	} else if b == 31 {
		return d.decodeVarUint()
	} else {
		return 0, fmt.Errorf("invalid value length at pos:%d value 0x%x", d.offset()-1, b)
	}
}

func (d *decoder) decodeVarUint() (uint32, error) {
	var (
		result  uint32
		more    = true
		maxIter = 5
	)
	for i := 0; i < maxIter && more; i++ {
		b, err := d.r.ReadByte()
		if err != nil {
			return 0, err
		}

		result <<= 7
		result |= uint32(b & 0x7f)
		more = b&0x80 == 0x80
	}
	if more {
		return 0, fmt.Errorf("invalid var uint")
	}
	return result, nil
}

func (d *decoder) decodeFrom() (string, error) {
	// From-value = Value-length (Address-present-token Encoded-string-value | Insert-address-token )
	// Address-present-token = <Octet 128>
	// Insert-address-token = <Octet 129>
	l, err := d.decodeValueLength()
	if err != nil {
		return "", err
	}
	if l < 1 {
		return "", fmt.Errorf("invalid from field")
	}

	buf := make([]byte, int(l))

	_, err = io.ReadFull(d.r, buf)
	if err != nil {
		return "", err
	}

	b := buf[0]

	switch b {
	case 128:
		rr := bytes.NewReader(buf[1:])

		tmpDecoder := decoder{
			r:      bufio.NewReader(rr),
			seeker: rr,
		}
		return tmpDecoder.decodeEncodedString()
	case 129:
		return "<insert-address-token>", nil
	}

	return "", fmt.Errorf("invalid from field token state: 0x%x", b)
}

func (d *decoder) decodeTextEnc() (string, error) {
	// Text-string = [Quote] *TEXT End-of-string
	// ; If the first character in the TEXT is in the range of 128-255, a Quote character must precede it.
	// ; Otherwise the Quote character must be omitted. The Quote is not part of the contents.

	peekbuf, err := d.r.Peek(1)
	if err != nil {
		return "", err
	}

	b := peekbuf[0]

	if b > 127 {
		// hasQuote
		d.r.ReadByte()
	}

	text, err := d.r.ReadBytes(0)
	if err != nil {
		return "", err
	}
	if len(text) <= 1 {
		return "", nil
	}

	return string(text[:len(text)-1]), nil
}

func (d *decoder) decodeConstrainedMedia() (string, error) {
	peakbuf, err := d.r.Peek(1)
	if err != nil {
		return "", err
	}
	b := peakbuf[0]
	if b < 127 {
		// Extension-Media = *TEXT End-of-string
		// *TEXT = byte array where each byte > 31 < 127
		text, err := d.r.ReadBytes(0)
		if err != nil {
			return "", err
		}
		if len(text) <= 1 {
			return "", nil
		}
		return string(text[len(text)-1]), nil
	} else {
		b, err = d.decodeShortInt()
		if err != nil {
			return "", err
		}
		if int(b) >= len(contentTypes) {
			return "", fmt.Errorf("unknown short content type %d", b)
		}
		return contentTypes[b], nil
	}
}

func (d *decoder) decodeMessageType() (HeaderMessageType, error) {
	// 	7.2.14. Message-Type field
	// Message-type-value = m-send-req | m-send-conf | m-notification-ind | m-notifyresp-ind | m-retrieve-conf | macknowledge-ind | m-delivery-ind
	// m-send-req = <Octet 128>
	// m-send-conf = <Octet 129>
	// m-notification-ind = <Octet 130>
	// m-notifyresp-ind = <Octet 131>
	// m-retrieve-conf = <Octet 132>
	// m-acknowledge-ind = <Octet 133>
	// m-delivery-ind = <Octet 134>
	// Unknown message types will be discarded.

	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}

	if b < 128 || b > 134 {
		return UnknownMessageType, nil
	}

	return HeaderMessageType(b), nil
}

func (d *decoder) decodeVersion() (string, error) {
	//MMS-version-value = Short-integer
	// The three most significant bits of the Short-integer are interpreted to encode a major version number in the range 1-7,
	// and the four least significant bits contain a minor version number in the range 0-14. If there is only a major version
	// number, this is encoded by placing the value 15 in the four least significant bits [WAPWSP].

	b, err := d.decodeShortInt()
	if err != nil {
		return "", err
	}

	major := (b & 0x70) >> 4
	minor := b & 0x0f
	if minor == 15 {
		minor = 0
	}
	return fmt.Sprintf("%d.%d", major, minor), nil
}

func (d *decoder) decodePriority() (HeaderPriority, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	return HeaderPriority(b), nil
}

func (d *decoder) decodeResponseStatus() (HeaderResponseStatus, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	return HeaderResponseStatus(b), nil
}

func (d *decoder) decodeSenderVisibility() (HederSenderVisibility, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	return HederSenderVisibility(b), nil
}

func (d *decoder) decodeStatus() (HeaderStatus, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return 0, err
	}
	return HeaderStatus(b), nil
}

type MMSField int

const (
	Bcc              MMSField = 0x01
	Cc               MMSField = 0x02
	ContentLocation  MMSField = 0x03
	ContentType      MMSField = 0x04
	Date             MMSField = 0x05
	DeliveryReport   MMSField = 0x06
	DeliveryTime     MMSField = 0x07
	Expiry           MMSField = 0x08
	From             MMSField = 0x09
	MessageClass     MMSField = 0x0a
	MessageID        MMSField = 0x0b
	MessageType      MMSField = 0x0c
	MMSVersion       MMSField = 0x0d
	MessageSize      MMSField = 0x0e
	Priority         MMSField = 0x0f
	ReadReply        MMSField = 0x10
	ReportAllowed    MMSField = 0x11
	ResponseStatus   MMSField = 0x12
	ResponseText     MMSField = 0x13
	SenderVisibility MMSField = 0x14
	StatusField      MMSField = 0x15
	Subject          MMSField = 0x16
	To               MMSField = 0x17
	TransactionID    MMSField = 0x18

	RetrieveStatus         MMSField = 0x19
	RetrieveText           MMSField = 0x20
	ReadStatus             MMSField = 0x21
	ReplayCharging         MMSField = 0x22
	ReplayChargingDeadline MMSField = 0x23
	ReplayChargingID       MMSField = 0x24
	ReplayChargingSize     MMSField = 0x25
)

func (f MMSField) String() string {
	switch f {
	case Bcc:
		return "Bcc"
	case Cc:
		return "Cc"
	case ContentLocation:
		return "Content-Location"
	case ContentType:
		return "Content-Type"
	case Date:
		return "Date"
	case DeliveryReport:
		return "Delivery-Report"
	case DeliveryTime:
		return "Delivery-Time"
	case Expiry:
		return "Expiry"
	case From:
		return "From"
	case MessageClass:
		return "Message-Class"
	case MessageID:
		return "Message-ID"
	case MessageType:
		return "Message-Type"
	case MMSVersion:
		return "MMS-Version"
	case MessageSize:
		return "Message-Size"
	case Priority:
		return "Priority"
	case ReadReply:
		return "Read-Reply"
	case ReportAllowed:
		return "Report-Allowed"
	case ResponseStatus:
		return "Response-Status"
	case ResponseText:
		return "Response-Text"
	case SenderVisibility:
		return "Sender-Visibility"
	case StatusField:
		return "Status"
	case Subject:
		return "Subject"
	case To:
		return "To"
	case TransactionID:
		return "Transaction-ID"
	case RetrieveStatus:
		return "Retrieve-Status"
	case RetrieveText:
		return "Retrieve-Text"
	case ReadStatus:
		return "Read-Status"
	case ReplayCharging:
		return "Replay-Charging"
	case ReplayChargingDeadline:
		return "Replay-Charging-Deadline"
	case ReplayChargingID:
		return "Replay-Charging-ID"
	case ReplayChargingSize:
		return "Replay-Charging-Size"

	default:
		return fmt.Sprintf("UnknownMMSField<%d>", f)
	}
}
