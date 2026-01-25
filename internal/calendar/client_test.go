package calendar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientStructure(t *testing.T) {
	t.Run("Client has private service field", func(t *testing.T) {
		client := &Client{}
		assert.Nil(t, client.service)
	})
}
