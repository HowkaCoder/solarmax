package store

import (
	"encoding/json"
	"time"
)

// Snapshot отдаёт полный дамп хранилища в JSON.
// Нужен, чтобы на бесплатном хостинге (где диск может быть временным)
// можно было в любой момент забрать данные к себе и потом залить обратно.
func (s *Store) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.d.SavedAt = time.Now().UTC()
	return json.MarshalIndent(s.d, "", "  ")
}

// Restore полностью заменяет содержимое хранилища дампом из Snapshot.
func (s *Store) Restore(raw []byte) error {
	var loaded data
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return badRequest("некорректный формат дампа: " + err.Error())
	}
	loaded.init()

	s.mu.Lock()
	s.d = &loaded
	s.mu.Unlock()

	s.markDirty()
	return s.Save()
}
