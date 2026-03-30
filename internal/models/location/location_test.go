package location

import "testing"

func TestLocationMapURLs(t *testing.T) {
	latitude := 19.432608
	longitude := -99.133209
	loc := &Location{
		Latitude:  &latitude,
		Longitude: &longitude,
	}

	link := loc.MapLinkURL()
	if want := "https://www.openstreetmap.org/?mlat=19.432608&mlon=-99.133209#map=13/19.432608/-99.133209"; link != want {
		t.Fatalf("expected map link %q, got %q", want, link)
	}

	embed := loc.MapEmbedURL()
	if want := "https://www.openstreetmap.org/export/embed.html?bbox=-99.153209,19.412608,-99.113209,19.452608&layer=mapnik&marker=19.432608,-99.133209"; embed != want {
		t.Fatalf("expected embed url %q, got %q", want, embed)
	}
}

func TestLocationMapURLsRequireCoordinates(t *testing.T) {
	if got := (&Location{}).MapLinkURL(); got != "" {
		t.Fatalf("expected empty map link without coordinates, got %q", got)
	}

	if got := (&Location{}).MapEmbedURL(); got != "" {
		t.Fatalf("expected empty embed url without coordinates, got %q", got)
	}
}
