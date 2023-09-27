package apollo

import (
	"encoding/json"
	"sync"
)

type notify struct {
	notifications map[string]int
	lock          sync.RWMutex
}

type notification struct {
	NamespaceName  string `json:"namespaceName,omitempty"`
	NotificationID int    `json:"notificationId,omitempty"`
}

func (n *notify) getNotifyString() string {
	n.lock.RLock()
	defer n.lock.RUnlock()
	var list []*notification
	for k, v := range n.notifications {
		list = append(list, &notification{
			k, v,
		})
	}
	bts, err := json.Marshal(&list)
	if err != nil {
		return ""
	}
	return string(bts)
}

func (n *notify) put(key string, value int) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.notifications[key] = value
}
