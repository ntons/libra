package database

type nopFilter struct{}

func (f nopFilter) Encode(in []byte) ([]byte, bool, error) {
	return in, false, nil
}
func (f nopFilter) Decode(in []byte) ([]byte, error) {
	return in, nil
}
