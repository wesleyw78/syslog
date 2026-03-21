package handlers

import (
	"net/http"

	"syslog/internal/domain"
	"syslog/internal/repository"
)

type logEntry struct {
	Message domain.SyslogMessage `json:"message"`
	Event   *domain.ClientEvent  `json:"event,omitempty"`
}

func NewLogsHandler(messagesRepo repository.SyslogMessageRepository, eventsRepo repository.ClientEventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if messagesRepo == nil || eventsRepo == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		messages, err := messagesRepo.ListRecent(r.Context(), 20)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		events, err := eventsRepo.ListRecent(r.Context(), 20)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		eventsByMessageID := make(map[uint64]domain.ClientEvent, len(events))
		for _, event := range events {
			eventsByMessageID[event.SyslogMessageID] = event
		}

		items := make([]any, 0, len(messages))
		for _, message := range messages {
			entry := logEntry{Message: message}
			if event, ok := eventsByMessageID[message.ID]; ok {
				copied := event
				entry.Event = &copied
			}
			items = append(items, entry)
		}

		writeJSON(w, http.StatusOK, listResponse{Items: items})
	}
}
