package contacts

import (
	"testing"
)

func TestClientStructure(t *testing.T) {
	t.Run("Client has private service field", func(t *testing.T) {
		client := &Client{}
		if client.service != nil {
			t.Errorf("got %v, want nil", client.service)
		}
	})
}
