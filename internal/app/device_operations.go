package app

import (
	"context"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
)

type DeviceOperations struct {
	devices *store.DeviceStore
	now     func() time.Time
}

func NewDeviceOperations(devices *store.DeviceStore, now func() time.Time) *DeviceOperations {
	return &DeviceOperations{devices: devices, now: now}
}

func (operations *DeviceOperations) ListDevices(ctx context.Context, principal identity.Principal) ([]identity.Device, error) {
	return operations.devices.List(ctx, principal.Account.ID)
}

func (operations *DeviceOperations) RevokeDevice(ctx context.Context, principal identity.Principal, deviceID string) error {
	return operations.devices.Revoke(ctx, principal.Account.ID, deviceID, operations.now().UTC())
}

var _ httpapi.DeviceOperations = (*DeviceOperations)(nil)
