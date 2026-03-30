package locations

import "testing"

func TestParseCoordinates(t *testing.T) {
	t.Run("accepts valid coordinates", func(t *testing.T) {
		latitude, longitude, err := parseCoordinates("19.432608", "-99.133209")
		if err != nil {
			t.Fatalf("expected valid coordinates, got error: %v", err)
		}
		if latitude == nil || longitude == nil {
			t.Fatal("expected parsed coordinates")
		}
	})

	t.Run("rejects swapped coordinates with explicit hint", func(t *testing.T) {
		_, _, err := parseCoordinates("102", "-101")
		if err == nil {
			t.Fatal("expected swapped coordinates to fail")
		}
		if got := err.Error(); got != "Latitude must be a number between -90 and 90. Enter latitude first and longitude second." &&
			got != "Latitude must be between -90 and 90. It looks like you may have swapped the fields: enter latitude first and longitude second, for example 19.432608 and -99.133209." {
			t.Fatalf("unexpected error: %q", got)
		}
	})
}
