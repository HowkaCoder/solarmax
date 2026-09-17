package store

import (
	"sort"
	"time"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

// ==================== УСЛУГИ ====================

func (s *Store) ListServices(status, language string) []models.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Service{}
	for _, sv := range values(&s.d.Services) {
		if status != "" && sv.Status != status {
			continue
		}
		if !hasLang(sv.Language, language) {
			continue
		}
		out := *sv
		out.Images = s.mediaFor("service", sv.ID)
		list = append(list, out)
	}

	sort.Slice(list, func(i, j int) bool { return list[i].ID > list[j].ID })
	return list
}

func (s *Store) GetService(id int64) (models.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sv, ok := s.d.Services.Items[id]
	if !ok {
		return models.Service{}, notFound("услуга не найдена")
	}
	out := *sv
	out.Images = s.mediaFor("service", id)
	return out, nil
}

func (s *Store) CreateService(in models.ServiceInput) (models.Service, error) {
	if in.Name == "" {
		return models.Service{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Service{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ServiceTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sv := range s.d.Services.Items {
		if sv.Slug == in.Slug {
			return models.Service{}, conflict("услуга с таким slug уже существует")
		}
	}

	now := time.Now().UTC()
	sv := &models.Service{
		ID:        s.d.Services.nextID(),
		Name:      in.Name,
		Slug:      in.Slug,
		Status:    status,
		Language:  in.Language,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.d.Services.Items[sv.ID] = sv
	s.markDirty()

	return *sv, nil
}

func (s *Store) UpdateService(id int64, in models.ServiceInput) (models.Service, error) {
	if in.Name == "" {
		return models.Service{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Service{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ServiceTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Services.Items[id]
	if !ok {
		return models.Service{}, notFound("услуга не найдена")
	}
	for _, sv := range s.d.Services.Items {
		if sv.ID != id && sv.Slug == in.Slug {
			return models.Service{}, conflict("услуга с таким slug уже существует")
		}
	}

	updated := &models.Service{
		ID:        id,
		Name:      in.Name,
		Slug:      in.Slug,
		Status:    status,
		Language:  in.Language,
		CreatedAt: old.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
	s.d.Services.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("service", id)
	return out, nil
}

func (s *Store) SetServiceStatus(id int64, status string) error {
	if status != "active" && status != "inactive" {
		return badRequest("status должен быть active или inactive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Services.Items[id]
	if !ok {
		return notFound("услуга не найдена")
	}
	updated := *old
	updated.Status = status
	updated.UpdatedAt = time.Now().UTC()
	s.d.Services.Items[id] = &updated
	s.markDirty()
	return nil
}

func (s *Store) DeleteService(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Services.Items[id]; !ok {
		return notFound("услуга не найдена")
	}
	delete(s.d.Services.Items, id)
	s.deleteMediaForEntity("service", id)
	s.markDirty()
	return nil
}

// ==================== ТИПЫ ОБЪЕКТОВ ====================

func (s *Store) ListObjectTypes(status, language string) []models.ObjectType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.ObjectType{}
	for _, o := range values(&s.d.ObjectTypes) {
		if status != "" && o.Status != status {
			continue
		}
		if !hasLang(o.Language, language) {
			continue
		}
		out := *o
		out.Images = s.mediaFor("object_type", o.ID)
		list = append(list, out)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Store) GetObjectType(id int64) (models.ObjectType, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	o, ok := s.d.ObjectTypes.Items[id]
	if !ok {
		return models.ObjectType{}, notFound("объект не найден")
	}
	out := *o
	out.Images = s.mediaFor("object_type", id)
	return out, nil
}

func (s *Store) CreateObjectType(in models.ObjectTypeInput) (models.ObjectType, error) {
	if in.Name == "" {
		return models.ObjectType{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.ObjectType{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ObjectTypeTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, o := range s.d.ObjectTypes.Items {
		if o.Slug == in.Slug {
			return models.ObjectType{}, conflict("объект с таким slug уже существует")
		}
	}

	now := time.Now().UTC()
	o := &models.ObjectType{
		ID:        s.d.ObjectTypes.nextID(),
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.d.ObjectTypes.Items[o.ID] = o
	s.markDirty()

	return *o, nil
}

func (s *Store) UpdateObjectType(id int64, in models.ObjectTypeInput) (models.ObjectType, error) {
	if in.Name == "" {
		return models.ObjectType{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.ObjectType{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ObjectTypeTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.ObjectTypes.Items[id]
	if !ok {
		return models.ObjectType{}, notFound("объект не найден")
	}
	for _, o := range s.d.ObjectTypes.Items {
		if o.ID != id && o.Slug == in.Slug {
			return models.ObjectType{}, conflict("объект с таким slug уже существует")
		}
	}

	updated := &models.ObjectType{
		ID:        id,
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: old.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
	s.d.ObjectTypes.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("object_type", id)
	return out, nil
}

func (s *Store) SetObjectTypeStatus(id int64, status string) error {
	if status != "active" && status != "inactive" {
		return badRequest("status должен быть active или inactive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.ObjectTypes.Items[id]
	if !ok {
		return notFound("объект не найден")
	}
	updated := *old
	updated.Status = status
	updated.UpdatedAt = time.Now().UTC()
	s.d.ObjectTypes.Items[id] = &updated
	s.markDirty()
	return nil
}

func (s *Store) DeleteObjectType(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.ObjectTypes.Items[id]; !ok {
		return notFound("объект не найден")
	}
	delete(s.d.ObjectTypes.Items, id)
	s.deleteMediaForEntity("object_type", id)
	s.markDirty()
	return nil
}

// ==================== ПРЕИМУЩЕСТВА ====================

func (s *Store) ListAdvantages(status, language string) []models.Advantage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Advantage{}
	for _, a := range values(&s.d.Advantages) {
		if status != "" && a.Status != status {
			continue
		}
		if !hasLang(a.Language, language) {
			continue
		}
		out := *a
		out.Images = s.mediaFor("advantage", a.ID)
		list = append(list, out)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].SortOrder != list[j].SortOrder {
			return list[i].SortOrder < list[j].SortOrder
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Store) GetAdvantage(id int64) (models.Advantage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.d.Advantages.Items[id]
	if !ok {
		return models.Advantage{}, notFound("преимущество не найдено")
	}
	out := *a
	out.Images = s.mediaFor("advantage", id)
	return out, nil
}

func (s *Store) CreateAdvantage(in models.AdvantageInput) (models.Advantage, error) {
	if in.Name == "" {
		return models.Advantage{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Advantage{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.AdvantageTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, a := range s.d.Advantages.Items {
		if a.Slug == in.Slug {
			return models.Advantage{}, conflict("преимущество с таким slug уже существует")
		}
	}

	now := time.Now().UTC()
	a := &models.Advantage{
		ID:        s.d.Advantages.nextID(),
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.d.Advantages.Items[a.ID] = a
	s.markDirty()

	return *a, nil
}

func (s *Store) UpdateAdvantage(id int64, in models.AdvantageInput) (models.Advantage, error) {
	if in.Name == "" {
		return models.Advantage{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Advantage{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.AdvantageTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Advantages.Items[id]
	if !ok {
		return models.Advantage{}, notFound("преимущество не найдено")
	}
	for _, a := range s.d.Advantages.Items {
		if a.ID != id && a.Slug == in.Slug {
			return models.Advantage{}, conflict("преимущество с таким slug уже существует")
		}
	}

	updated := &models.Advantage{
		ID:        id,
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: old.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
	s.d.Advantages.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("advantage", id)
	return out, nil
}

func (s *Store) SetAdvantageStatus(id int64, status string) error {
	if status != "active" && status != "inactive" {
		return badRequest("status должен быть active или inactive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Advantages.Items[id]
	if !ok {
		return notFound("преимущество не найдено")
	}
	updated := *old
	updated.Status = status
	updated.UpdatedAt = time.Now().UTC()
	s.d.Advantages.Items[id] = &updated
	s.markDirty()
	return nil
}

func (s *Store) DeleteAdvantage(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Advantages.Items[id]; !ok {
		return notFound("преимущество не найдено")
	}
	delete(s.d.Advantages.Items, id)
	s.deleteMediaForEntity("advantage", id)
	s.markDirty()
	return nil
}
