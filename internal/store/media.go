package store

import (
	"sort"
	"time"

	"solarmax/internal/models"
)

var allowedEntityTypes = map[string]bool{
	"category":    true,
	"subcategory": true,
	"product":     true,
	"service":     true,
	"object_type": true,
	"advantage":   true,
}

// IsKnownEntityType - проверка значения поля entity_type при загрузке файлов.
func IsKnownEntityType(t string) bool { return allowedEntityTypes[t] }

func (s *Store) ListMedia(entityType string, entityID int64) []models.Media {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := s.mediaFor(entityType, entityID)
	if list == nil {
		return []models.Media{}
	}
	return list
}

// mediaFor возвращает картинки сущности: главная первой, далее по sort_order.
// Вызывается под уже взятым локом.
func (s *Store) mediaFor(entityType string, entityID int64) []models.Media {
	var list []models.Media
	for _, m := range s.d.Media.Items {
		if m.EntityType == entityType && m.EntityID == entityID {
			list = append(list, *m)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].IsMain != list[j].IsMain {
			return list[i].IsMain
		}
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return list[i].ID < list[j].ID
	})
	return list
}

// AddMedia регистрирует уже сохранённый на диск файл.
func (s *Store) AddMedia(entityType string, entityID int64, url string, isMain bool, sortOrder int) (models.Media, error) {
	if !allowedEntityTypes[entityType] {
		return models.Media{}, badRequest("недопустимый entity_type")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m := &models.Media{
		ID:         s.d.Media.nextID(),
		EntityType: entityType,
		EntityID:   entityID,
		URL:        url,
		IsMain:     isMain,
		SortOrder:  sortOrder,
		CreatedAt:  time.Now().UTC(),
	}
	s.d.Media.Items[m.ID] = m

	// Главная картинка у сущности всегда одна.
	if isMain {
		s.unsetMainExcept(entityType, entityID, m.ID)
	}
	s.markDirty()

	return *m, nil
}

// DeleteMedia убирает запись и возвращает её URL, чтобы хендлер удалил файл с диска.
func (s *Store) DeleteMedia(id int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.d.Media.Items[id]
	if !ok {
		return "", notFound("медиафайл не найден")
	}
	delete(s.d.Media.Items, id)
	s.markDirty()
	return m.URL, nil
}

// SetMainMedia делает картинку главной и возвращает обновлённый список сущности.
func (s *Store) SetMainMedia(id int64) ([]models.Media, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.d.Media.Items[id]
	if !ok {
		return nil, notFound("медиафайл не найден")
	}

	updated := *m
	updated.IsMain = true
	s.d.Media.Items[id] = &updated
	s.unsetMainExcept(m.EntityType, m.EntityID, id)
	s.markDirty()

	return s.mediaFor(m.EntityType, m.EntityID), nil
}

func (s *Store) unsetMainExcept(entityType string, entityID, exceptID int64) {
	for id, m := range s.d.Media.Items {
		if id == exceptID || m.EntityType != entityType || m.EntityID != entityID || !m.IsMain {
			continue
		}
		updated := *m
		updated.IsMain = false
		s.d.Media.Items[id] = &updated
	}
}

// deleteMediaForEntity вызывается при удалении сущности. Вызывается под локом.
func (s *Store) deleteMediaForEntity(entityType string, entityID int64) {
	for id, m := range s.d.Media.Items {
		if m.EntityType == entityType && m.EntityID == entityID {
			delete(s.d.Media.Items, id)
		}
	}
}

// ==================== АДМИНЫ ====================

// EnsureAdmin создаёт администратора, если в хранилище ещё нет ни одного.
// Возвращает true, если пользователь был создан.
func (s *Store) EnsureAdmin(username, passwordHash string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.d.Admins.Items) > 0 {
		return false, nil
	}

	now := time.Now().UTC()
	a := &storedAdmin{
		ID:           s.d.Admins.nextID(),
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.d.Admins.Items[a.ID] = a
	s.markDirty()
	return true, nil
}

// PasswordHash возвращает хэш пароля администратора по логину.
func (s *Store) PasswordHash(username string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, a := range s.d.Admins.Items {
		if a.Username == username {
			return a.PasswordHash, true
		}
	}
	return "", false
}

// SetPasswordHash меняет пароль администратора.
func (s *Store) SetPasswordHash(username, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, a := range s.d.Admins.Items {
		if a.Username == username {
			updated := *a
			updated.PasswordHash = hash
			updated.UpdatedAt = time.Now().UTC()
			s.d.Admins.Items[id] = &updated
			s.markDirty()
			return nil
		}
	}
	return notFound("пользователь не найден")
}
