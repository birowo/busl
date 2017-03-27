package broker

import (
	"testing"

	"github.com/heroku/busl/util"
	"github.com/stretchr/testify/assert"
)

func newRegUUID() (*RedisRegistrar, string) {
	reg := NewRedisRegistrar()
	uuid, _ := util.NewUUID()

	return reg, uuid
}

func TestRegisteredIsRegistered(t *testing.T) {
	reg, uuid := newRegUUID()
	reg.Register(uuid)

	r, err := reg.IsRegistered(uuid)
	assert.Nil(t, err)
	assert.True(t, r)
}

func TestUnregisteredIsNotRegistered(t *testing.T) {
	reg, uuid := newRegUUID()
	r, err := reg.IsRegistered(uuid)
	assert.Nil(t, err)
	assert.False(t, r)
}

func TestUnregisteredErrNotRegistered(t *testing.T) {
	_, uuid := newRegUUID()

	_, err := NewReader(uuid)
	assert.Equal(t, err, ErrNotRegistered)

	_, err = NewWriter(uuid)
	assert.Equal(t, err, ErrNotRegistered)
}

func TestRegisteredNoError(t *testing.T) {
	reg, uuid := newRegUUID()
	reg.Register(uuid)
	_, err := NewReader(uuid)
	assert.Nil(t, err)

	_, err = NewWriter(uuid)
	assert.Nil(t, err)
}
