package daemon

import "testing"

func TestGetKeyInfo(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		wantKey     string
		wantCode    string
		wantKeyCode int
		wantText    string
	}{
		{
			name:        "Enter key has text for keypress",
			key:         "Enter",
			wantKey:     "Enter",
			wantCode:    "Enter",
			wantKeyCode: 13,
			wantText:    "\r",
		},
		{
			name:        "Tab has no text",
			key:         "Tab",
			wantKey:     "Tab",
			wantCode:    "Tab",
			wantKeyCode: 9,
			wantText:    "",
		},
		{
			name:        "Escape has no text",
			key:         "Escape",
			wantKey:     "Escape",
			wantCode:    "Escape",
			wantKeyCode: 27,
			wantText:    "",
		},
		{
			name:        "Backspace has no text",
			key:         "Backspace",
			wantKey:     "Backspace",
			wantCode:    "Backspace",
			wantKeyCode: 8,
			wantText:    "",
		},
		{
			name:        "lowercase letter",
			key:         "a",
			wantKey:     "a",
			wantCode:    "KeyA",
			wantKeyCode: 65,
			wantText:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := getKeyInfo(tt.key)

			if info.key != tt.wantKey {
				t.Errorf("key = %q, want %q", info.key, tt.wantKey)
			}
			if info.code != tt.wantCode {
				t.Errorf("code = %q, want %q", info.code, tt.wantCode)
			}
			if info.keyCode != tt.wantKeyCode {
				t.Errorf("keyCode = %d, want %d", info.keyCode, tt.wantKeyCode)
			}
			if info.text != tt.wantText {
				t.Errorf("text = %q, want %q", info.text, tt.wantText)
			}
		})
	}
}
