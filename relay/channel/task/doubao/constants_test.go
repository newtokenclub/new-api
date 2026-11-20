package doubao

import (
	"math"
	"testing"
)

func TestModelListIncludesSeedance25(t *testing.T) {
	const model = "doubao-seedance-2-5-260628"
	for _, candidate := range ModelList {
		if candidate == model {
			return
		}
	}
	t.Fatalf("ModelList does not include %q", model)
}

func TestSeedance25VideoInputRatio(t *testing.T) {
	const model = "doubao-seedance-2-5-260628"
	tests := []struct {
		name     string
		hasVideo bool
		want     float64
	}{
		{name: "without video", hasVideo: false, want: 1.0},
		{name: "with video", hasVideo: true, want: 0.6},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := GetVideoInputRatio(model, "720p", test.hasVideo)
			if !ok {
				t.Fatal("expected Seedance 2.5 pricing configuration")
			}
			if math.Abs(got-test.want) > 1e-9 {
				t.Fatalf("GetVideoInputRatio() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSeedance25FourKVideoInputRatio(t *testing.T) {
	const model = "doubao-seedance-2-5-260628"
	for _, test := range []struct {
		name     string
		hasVideo bool
		want     float64
	}{
		{name: "without reference video", hasVideo: false, want: 31.0 / 70.0},
		{name: "with reference video", hasVideo: true, want: 19.0 / 70.0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := GetVideoInputRatio(model, "4k", test.hasVideo)
			if !ok {
				t.Fatal("expected Seedance 2.5 pricing configuration")
			}
			if math.Abs(got-test.want) > 1e-9 {
				t.Fatalf("GetVideoInputRatio() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSeedance25FlashVideoInputRatio(t *testing.T) {
	const model = "doubao-seedance-2-5-flash-261115"
	found := false
	for _, candidate := range ModelList {
		if candidate == model {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ModelList does not include %q", model)
	}

	for _, resolution := range []string{"480p", "720p"} {
		t.Run(resolution, func(t *testing.T) {
			for _, test := range []struct {
				name     string
				hasVideo bool
				want     float64
			}{
				{name: "without video", hasVideo: false, want: 1.0},
				{name: "with video", hasVideo: true, want: 28.0 / 46.0},
			} {
				t.Run(test.name, func(t *testing.T) {
					got, ok := GetVideoInputRatio(model, resolution, test.hasVideo)
					if !ok {
						t.Fatal("expected Seedance 2.5 Flash pricing configuration")
					}
					if math.Abs(got-test.want) > 1e-9 {
						t.Fatalf("GetVideoInputRatio() = %v, want %v", got, test.want)
					}
				})
			}
		})
	}

	// Other resolutions retain the existing base-price fallback; the upstream
	// service is responsible for rejecting unsupported values.
	if got, ok := GetVideoInputRatio(model, "1080p", true); !ok || got != 1.0 {
		t.Fatalf("1080p fallback = (%v, %v), want (1, true)", got, ok)
	}
}

func TestSeedance20MiniVideoInputRatio(t *testing.T) {
	const model = "doubao-seedance-2-0-mini-260615"
	found := false
	for _, candidate := range ModelList {
		if candidate == model {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ModelList does not include %q", model)
	}

	tests := []struct {
		name     string
		hasVideo bool
		want     float64
	}{
		{name: "without reference video", hasVideo: false, want: 1.0},
		{name: "with reference video", hasVideo: true, want: 14.0 / 23.0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := GetVideoInputRatio(model, "720p", test.hasVideo)
			if !ok {
				t.Fatal("expected Seedance 2.0 mini pricing configuration")
			}
			if math.Abs(got-test.want) > 1e-9 {
				t.Fatalf("GetVideoInputRatio() = %v, want %v", got, test.want)
			}
		})
	}
}
