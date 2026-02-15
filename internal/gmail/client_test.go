package gmail

import (
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestGetLabelName(t *testing.T) {
	t.Run("returns name for cached label", func(t *testing.T) {
		client := &Client{
			labels: map[string]*gmailapi.Label{
				"Label_123": {Id: "Label_123", Name: "Work"},
				"Label_456": {Id: "Label_456", Name: "Personal"},
			},
			labelsLoaded: true,
		}

		if got := client.GetLabelName("Label_123"); got != "Work" {
			t.Errorf("got %v, want %v", got, "Work")
		}
		if got := client.GetLabelName("Label_456"); got != "Personal" {
			t.Errorf("got %v, want %v", got, "Personal")
		}
	})

	t.Run("returns ID for uncached label", func(t *testing.T) {
		client := &Client{
			labels:       map[string]*gmailapi.Label{},
			labelsLoaded: true,
		}

		if got := client.GetLabelName("Unknown_Label"); got != "Unknown_Label" {
			t.Errorf("got %v, want %v", got, "Unknown_Label")
		}
	})

	t.Run("returns ID when labels not loaded", func(t *testing.T) {
		client := &Client{
			labels:       nil,
			labelsLoaded: false,
		}

		if got := client.GetLabelName("Label_123"); got != "Label_123" {
			t.Errorf("got %v, want %v", got, "Label_123")
		}
	})
}

func TestGetLabels(t *testing.T) {
	t.Run("returns nil when labels not loaded", func(t *testing.T) {
		client := &Client{
			labels:       nil,
			labelsLoaded: false,
		}

		result := client.GetLabels()
		if result != nil {
			t.Errorf("got %v, want nil", result)
		}
	})

	t.Run("returns all cached labels", func(t *testing.T) {
		label1 := &gmailapi.Label{Id: "Label_1", Name: "Work"}
		label2 := &gmailapi.Label{Id: "Label_2", Name: "Personal"}

		client := &Client{
			labels: map[string]*gmailapi.Label{
				"Label_1": label1,
				"Label_2": label2,
			},
			labelsLoaded: true,
		}

		result := client.GetLabels()
		if len(result) != 2 {
			t.Errorf("got length %d, want %d", len(result), 2)
		}
		found1, found2 := false, false
		for _, l := range result {
			if l == label1 {
				found1 = true
			}
			if l == label2 {
				found2 = true
			}
		}
		if !found1 {
			t.Errorf("expected result to contain label1 (Work)")
		}
		if !found2 {
			t.Errorf("expected result to contain label2 (Personal)")
		}
	})

	t.Run("returns empty slice for empty cache", func(t *testing.T) {
		client := &Client{
			labels:       map[string]*gmailapi.Label{},
			labelsLoaded: true,
		}

		result := client.GetLabels()
		if result == nil {
			t.Fatal("expected non-nil, got nil")
		}
		if len(result) != 0 {
			t.Errorf("got length %d, want 0", len(result))
		}
	})
}
