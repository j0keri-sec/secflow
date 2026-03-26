package pusher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/secflow/server/internal/model"
)

func TestPusherFactory_CreateFromChannel(t *testing.T) {
	factory := NewPusherFactory()

	tests := []struct {
		name      string
		channel  *model.PushChannel
		wantErr  bool
		errType  error
	}{
		{
			name: "dingding channel",
			channel: &model.PushChannel{
				Type:   "dingding",
				Config: map[string]string{"access_token": "test_token", "sign_secret": "test_secret"},
			},
			wantErr: false,
		},
		{
			name: "lark channel",
			channel: &model.PushChannel{
				Type:   "lark",
				Config: map[string]string{"access_token": "test_token", "sign_secret": "test_secret"},
			},
			wantErr: false,
		},
		{
			name: "slack channel",
			channel: &model.PushChannel{
				Type:   "slack",
				Config: map[string]string{"webhook_url": "https://hooks.slack.com/test"},
			},
			wantErr: false,
		},
		{
			name: "telegram channel",
			channel: &model.PushChannel{
				Type:   "telegram",
				Config: map[string]string{"bot_token": "123456:test", "chat_id": "987654321"},
			},
			wantErr: false,
		},
		{
			name: "webhook channel",
			channel: &model.PushChannel{
				Type:   "webhook",
				Config: map[string]string{"url": "https://example.com/webhook", "method": "POST"},
			},
			wantErr: false,
		},
		{
			name: "unsupported channel",
			channel: &model.PushChannel{
				Type:   "unknown",
				Config: map[string]string{},
			},
			wantErr: true,
			errType: ErrUnsupportedChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pusher, err := factory.CreateFromChannel(tt.channel)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				assert.Nil(t, pusher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pusher)
			}
		})
	}
}

func TestPusherFactory_WithTimeout(t *testing.T) {
	factory := NewPusherFactory()
	assert.Equal(t, 10*time.Second, factory.timeout)

	customFactory := factory.WithTimeout(30 * time.Second)
	assert.Equal(t, 30*time.Second, customFactory.timeout)
	assert.Same(t, factory, customFactory) // Should return the same factory for chaining
}

func TestErrorTypes(t *testing.T) {
	// Test that error variables are defined correctly
	assert.Error(t, ErrUnsupportedChannel)
	assert.Error(t, ErrInvalidConfig)
	assert.Error(t, ErrPushFailed)

	// Test error messages
	assert.Contains(t, ErrUnsupportedChannel.Error(), "unsupported")
	assert.Contains(t, ErrInvalidConfig.Error(), "invalid")
	assert.Contains(t, ErrPushFailed.Error(), "push")
}

func TestRawMessage(t *testing.T) {
	msg := &RawMessage{
		Type:    "custom",
		Payload: map[string]string{"key": "value"},
	}
	assert.Equal(t, "custom", msg.Type)
	assert.Equal(t, map[string]string{"key": "value"}, msg.Payload)
}
