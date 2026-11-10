package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func tmpPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "settings.json")
}

func TestDefaultsWhenFileMissing(t *testing.T) {
	st, err := Open(tmpPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if st.Get() != Default() {
		t.Fatalf("got %+v", st.Get())
	}
}

func TestRoundTrip(t *testing.T) {
	path := tmpPath(t)
	st, _ := Open(path)

	want := Default()
	want.Theme = "nord"
	want.FontSize = 16
	want.AutoLockMinutes = 15
	if _, err := st.Set(want); err != nil {
		t.Fatal(err)
	}

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Get() != want {
		t.Fatalf("got %+v, want %+v", again.Get(), want)
	}
}

func TestSanitizeClampsEverything(t *testing.T) {
	got := Settings{
		Theme: "", FontFamily: "comic-sans", FontSize: 500, LineHeight: 0,
		CursorStyle: "zigzag", Scrollback: -5, AutoLockMinutes: 99999,
	}.Sanitize()

	want := Settings{
		Theme: "tokyo-night", FontFamily: "jetbrains-mono", FontSize: 28, LineHeight: 1.0,
		CursorStyle: "block", Scrollback: 500, AutoLockMinutes: 1440,
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPartialFileKeepsDefaults(t *testing.T) {
	path := tmpPath(t)
	os.WriteFile(path, []byte(`{"fontSize": 16}`), 0o600)

	st, _ := Open(path)
	got := st.Get()
	if got.FontSize != 16 || got.Theme != Default().Theme || got.Scrollback != Default().Scrollback {
		t.Fatalf("got %+v", got)
	}
}

func TestCorruptFileFallsBackToDefaults(t *testing.T) {
	path := tmpPath(t)
	os.WriteFile(path, []byte("{not json"), 0o600)

	st, err := Open(path)
	if err != nil {
		t.Fatalf("a corrupt settings file must not stop the app: %v", err)
	}
	if st.Get() != Default() {
		t.Fatalf("got %+v", st.Get())
	}
}