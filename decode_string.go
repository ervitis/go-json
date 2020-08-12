package json

import (
	"unsafe"
)

type stringDecoder struct {
}

func newStringDecoder() *stringDecoder {
	return &stringDecoder{}
}

func (d *stringDecoder) setDisallowUnknownFields(_ bool) {}

func (d *stringDecoder) decodeStream(s *stream, p uintptr) error {
	bytes, err := d.decodeStreamByte(s)
	if err != nil {
		return err
	}
	*(*string)(unsafe.Pointer(p)) = *(*string)(unsafe.Pointer(&bytes))
	return nil
}

func (d *stringDecoder) decode(buf []byte, cursor int64, p uintptr) (int64, error) {
	bytes, c, err := d.decodeByte(buf, cursor)
	if err != nil {
		return 0, err
	}
	cursor = c
	*(*string)(unsafe.Pointer(p)) = *(*string)(unsafe.Pointer(&bytes))
	return cursor, nil
}

func stringBytes(s *stream) ([]byte, error) {
	s.cursor++
	start := s.cursor
	for {
		switch s.char() {
		case '\\':
			s.cursor++
		case '"':
			literal := s.buf[start:s.cursor]
			s.cursor++
			s.reset()
			return literal, nil
		case nul:
			if s.read() {
				continue
			}
			goto ERROR
		}
		s.cursor++
	}
ERROR:
	return nil, errUnexpectedEndOfJSON("string", s.totalOffset())
}

func nullBytes(s *stream) error {
	if s.cursor+3 >= s.length {
		if !s.read() {
			return errInvalidCharacter(s.char(), "null", s.totalOffset())
		}
	}
	s.cursor++
	if s.char() != 'u' {
		return errInvalidCharacter(s.char(), "null", s.totalOffset())
	}
	s.cursor++
	if s.char() != 'l' {
		return errInvalidCharacter(s.char(), "null", s.totalOffset())
	}
	s.cursor++
	if s.char() != 'l' {
		return errInvalidCharacter(s.char(), "null", s.totalOffset())
	}
	s.cursor++
	return nil
}

func (d *stringDecoder) decodeStreamByte(s *stream) ([]byte, error) {
	for {
		switch s.char() {
		case ' ', '\n', '\t', '\r':
			s.cursor++
			continue
		case '"':
			return stringBytes(s)
		case 'n':
			if err := nullBytes(s); err != nil {
				return nil, err
			}
			return []byte{}, nil
		case nul:
			if s.read() {
				continue
			}
		}
		break
	}
	return nil, errNotAtBeginningOfValue(s.totalOffset())
}

func (d *stringDecoder) decodeByte(buf []byte, cursor int64) ([]byte, int64, error) {
	for {
		switch buf[cursor] {
		case ' ', '\n', '\t', '\r':
			cursor++
		case '"':
			cursor++
			start := cursor
			for {
				switch buf[cursor] {
				case '\\':
					cursor++
				case '"':
					literal := buf[start:cursor]
					cursor++
					return literal, cursor, nil
				case nul:
					return nil, 0, errUnexpectedEndOfJSON("string", cursor)
				}
				cursor++
			}
			return nil, 0, errUnexpectedEndOfJSON("string", cursor)
		case 'n':
			buflen := int64(len(buf))
			if cursor+3 >= buflen {
				return nil, 0, errUnexpectedEndOfJSON("null", cursor)
			}
			if buf[cursor+1] != 'u' {
				return nil, 0, errInvalidCharacter(buf[cursor+1], "null", cursor)
			}
			if buf[cursor+2] != 'l' {
				return nil, 0, errInvalidCharacter(buf[cursor+2], "null", cursor)
			}
			if buf[cursor+3] != 'l' {
				return nil, 0, errInvalidCharacter(buf[cursor+3], "null", cursor)
			}
			cursor += 4
			return []byte{}, cursor, nil
		default:
			goto ERROR
		}
	}
ERROR:
	return nil, 0, errNotAtBeginningOfValue(cursor)
}
