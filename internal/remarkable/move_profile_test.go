package remarkable

import "testing"

func TestMoveProfile(t *testing.T) {
	if moveScreenWidth != 954 || moveScreenHeight != 1696 {
		t.Fatalf("visible dimensions = %dx%d, want 954x1696", moveScreenWidth, moveScreenHeight)
	}
	if moveStorageWidth != 960 {
		t.Fatalf("storage width = %d, want 960", moveStorageWidth)
	}
	if moveStorageSizeBytes != 6_512_640 {
		t.Fatalf("storage size = %d, want 6512640", moveStorageSizeBytes)
	}
	if moveVisibleSizeBytes != 6_471_936 {
		t.Fatalf("visible size = %d, want 6471936", moveVisibleSizeBytes)
	}
	if moveMaxXValue != 6760 || moveMaxYValue != 11960 {
		t.Fatalf("pen ranges = %dx%d, want 6760x11960", moveMaxXValue, moveMaxYValue)
	}
}
