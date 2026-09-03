package comutils

import "encoding/hex"

func HexHasPrefix0x(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func HexTrim(hexStr string) string {
	if HexHasPrefix0x(hexStr) {
		return hexStr[2:]
	}
	return hexStr
}

func HexEncode(value []byte) string {
	return hex.EncodeToString(value)
}

func HexDecode(hexStr string) ([]byte, error) {
	return hex.DecodeString(HexTrim(hexStr))
}

func HexDecodeF(hexStr string) []byte {
	decodedBytes, err := HexDecode(hexStr)
	PanicOnError(err)
	return decodedBytes
}

func HexEncode0x(value []byte) string {
	return Hex0x + HexEncode(value)
}

func HexRandom(length int) (string, error) {
	data, err := RandomBytes(length)
	if err != nil {
		return "", err
	}
	return HexEncode(data), nil
}

func HexRandomF(length int) string {
	data, err := HexRandom(length)
	PanicOnError(err)
	return data
}
