package certificate

// GitCommit contains the hash for the git commit
//   export GIT_COMMIT=$(git rev-list -1 HEAD) && \
//     go build -ldflags "-X main.GitCommit=$GIT_COMMIT"
var GitCommit string

// Reader is an endless reader that writes the GitCommitMessage followed by nil bytes
type Reader struct {
	content []byte
	cur     int
}

func (r *Reader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		if r.cur >= len(r.content) {
			r.cur = 0
		}
		b[i] = []byte(r.content)[r.cur]
		r.cur++
	}
	return len(b), nil
}
