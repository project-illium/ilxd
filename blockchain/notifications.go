// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
)

// NotificationType represents the type of a notification message.
type NotificationType int

// NotificationCallback is used for a caller to provide a callback for
// notifications about various chain events.
//
// We use a callback rather than a channel here because we want to
// execute the callback with the state lock held to prevent races.
type NotificationCallback func(*Notification)

// Constants for the type of a notification message. We only have one now
// maybe we'll add others later so we put the plumbing here for it.
const (
	// NTBlockConnected indicates the associated block was connected to the chain.
	NTBlockConnected = iota
	NTAddValidator
	NTRemoveValidator
	NTNewEpoch
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTBlockConnected:  "NTBlockConnected",
	NTAddValidator:    "NTAddValidator",
	NTRemoveValidator: "NTRemoveValidator",
	NTNewEpoch:        "NTNewEpoch",
}

// String returns the NotificationType in human-readable form.
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

// Notification defines notification that is sent to the caller via the callback
// function provided during the call to New and consists of a notification type
// as well as associated data that depends on the type as follows:
//   - NTBlockConnected:    *blocks.Block
type Notification struct {
	Type NotificationType
	Data interface{}
}

// Subscribe to blockchain notifications. Registers a callback to be executed
// when various events take place. See the documentation on Notification and
// NotificationType for details on the types and contents of notifications.
func (b *Blockchain) Subscribe(callback NotificationCallback) {
	b.notificationsLock.Lock()
	b.notifications = append(b.notifications, callback)
	b.validatorSet.SubscribeEvents(callback)
	b.notificationsLock.Unlock()
}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
func (b *Blockchain) sendNotification(typ NotificationType, data interface{}) {
	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	b.notificationsLock.RLock()
	for _, callback := range b.notifications {
		go callback(&n)
	}
	b.notificationsLock.RUnlock()
}
