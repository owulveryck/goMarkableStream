package remarkable

import "testing"

func TestDeviceModelString(t *testing.T) {
	tests := []struct {
		model DeviceModel
		want  string
	}{
		{UnknownDevice, "UnknownDevice"},
		{Remarkable2, "Remarkable2"},
		{RemarkablePaperPro, "RemarkablePaperPro"},
		{RemarkablePaperProMove, "RemarkablePaperProMove"},
	}

	for _, test := range tests {
		if got := test.model.String(); got != test.want {
			t.Errorf("DeviceModel(%d).String() = %q, want %q", test.model, got, test.want)
		}
	}
}
