package certificate

import "testing"

func TestReader_Read(t *testing.T) {
	t.Run("buffer is bigger than content", func(t *testing.T) {
		rdr := &Reader{
			content: []byte(`abcd`),
		}
		b := make([]byte, 5)
		n, _ := rdr.Read(b)
		if n != 5 {
			t.Fail()
		}
		if string(b) != `abcda` {
			t.Errorf("Expected %s, got %s", "abcda", b)
		}
	})
	t.Run("buffer is bigger than contenti, 2 read", func(t *testing.T) {
		rdr := &Reader{
			content: []byte(`abcd`),
		}
		b := make([]byte, 5)
		n, _ := rdr.Read(b)
		if n != 5 {
			t.Fail()
		}
		if string(b) != `abcda` {
			t.Errorf("Expected %s, got %s", "abcda", b)
		}
		n, _ = rdr.Read(b)
		if n != 5 {
			t.Fail()
		}
		if string(b) != `bcdab` {
			t.Errorf("Expected %s, got %s", "bcdab", b)
		}
	})
	t.Run("buffer is smaller than content", func(t *testing.T) {
		rdr := &Reader{
			content: []byte(`abcd`),
		}
		b := make([]byte, 3)
		n, _ := rdr.Read(b)
		if n != 3 {
			t.Fail()
		}
		if string(b) != `abc` {
			t.Errorf("Expected %s, got %s", "abc", b)
		}
	})
	t.Run("buffer is smaller than content, 2 read", func(t *testing.T) {
		rdr := &Reader{
			content: []byte(`abcd`),
		}
		b := make([]byte, 3)
		n, _ := rdr.Read(b)
		if n != 3 {
			t.Fail()
		}
		if string(b) != `abc` {
			t.Errorf("Expected %s, got %s", "abc", b)
		}
		n, _ = rdr.Read(b)
		if n != 3 {
			t.Fail()
		}
		if string(b) != `dab` {
			t.Errorf("Expected %s, got %s", "dab", b)
		}
	})
}
