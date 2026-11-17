package jobs

const (
	minIdempotencyKeyBytes = 8
	maxIdempotencyKeyBytes = 128
)

func validIdempotencyKey(key string) bool {
	if len(key) < minIdempotencyKeyBytes || len(key) > maxIdempotencyKeyBytes {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] > 0x7f {
			return false
		}
	}
	return true
}
