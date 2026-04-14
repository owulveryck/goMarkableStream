//go:build !arm

package delta

// useHashEarlyExit controls whether xxhash.Sum64 is used for frame
// change detection. On amd64 and arm64, xxhash has optimized assembly
// and runs at ~7-8 GB/s, making the hash early-exit beneficial for
// idle frame detection.
const useHashEarlyExit = true
