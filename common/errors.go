package common

type (
	// BenchTestError represents an error that should abort / fail the whole bench test
	BenchTestError struct {
		Message string
	}
)

func (b *BenchTestError) Error() string {
	return b.Message
}
