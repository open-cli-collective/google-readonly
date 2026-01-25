package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientStructure(t *testing.T) {
	t.Run("Client has Service field", func(t *testing.T) {
		client := &Client{}
		assert.Nil(t, client.Service)
	})
}
