package store

import (
	"sort"
	"time"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

// ==================== КАТЕГОРИИ ====================

func (s *Store) ListCategories(status, language string) []models.Category {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Category{}
	for _, c := range values(&s.d.Categories) {
		if status != "" && c.Status != status {
			continue
		}
		if !hasLang(c.Language, language) {
			continue
		}
		out := *c
		out.Images = s.mediaFor("category", c.ID)
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

func (s *Store) GetCategory(id int64) (models.Category, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.d.Categories.Items[id]
	if !ok {
		return models.Category{}, notFound("категория не найдена")
	}
	out := *c
	out.Images = s.mediaFor("category", id)
	return out, nil
}

func (s *Store) CreateCategory(in models.CategoryInput) (models.Category, error) {
	if in.Name == "" {
		return models.Category{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Category{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.CategoryTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.categorySlugTaken(in.Slug, 0) {
		return models.Category{}, conflict("категория с таким slug уже существует")
	}

	now := time.Now().UTC()
	c := &models.Category{
		ID:        s.d.Categories.nextID(),
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.d.Categories.Items[c.ID] = c
	s.markDirty()

	return *c, nil
}

func (s *Store) UpdateCategory(id int64, in models.CategoryInput) (models.Category, error) {
	if in.Name == "" {
		return models.Category{}, badRequest("поле name обязательно")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Category{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.CategoryTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Categories.Items[id]
	if !ok {
		return models.Category{}, notFound("категория не найдена")
	}
	if s.categorySlugTaken(in.Slug, id) {
		return models.Category{}, conflict("категория с таким slug уже существует")
	}

	updated := &models.Category{
		ID:        id,
		Name:      in.Name,
		Slug:      in.Slug,
		SortOrder: in.SortOrder,
		Status:    status,
		Language:  in.Language,
		CreatedAt: old.CreatedAt,
		UpdatedAt: time.Now().UTC(),
	}
	s.d.Categories.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("category", id)
	return out, nil
}

func (s *Store) DeleteCategory(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Categories.Items[id]; !ok {
		return notFound("категория не найдена")
	}
	// Правило 1: нельзя удалить категорию, пока у неё есть подкатегории.
	for _, sub := range s.d.Subcategories.Items {
		if sub.CategoryID == id {
			return conflict("нельзя удалить категорию: у неё есть подкатегории")
		}
	}

	delete(s.d.Categories.Items, id)
	s.deleteMediaForEntity("category", id)
	s.markDirty()
	return nil
}

func (s *Store) categorySlugTaken(slug string, exceptID int64) bool {
	for _, c := range s.d.Categories.Items {
		if c.ID != exceptID && c.Slug == slug {
			return true
		}
	}
	return false
}

// ==================== ПОДКАТЕГОРИИ ====================

func (s *Store) ListSubcategories(status, language string, categoryID int64) []models.Subcategory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Subcategory{}
	for _, sub := range values(&s.d.Subcategories) {
		if status != "" && sub.Status != status {
			continue
		}
		if categoryID > 0 && sub.CategoryID != categoryID {
			continue
		}
		if !hasLang(sub.Language, language) {
			continue
		}
		out := *sub
		out.Images = s.mediaFor("subcategory", sub.ID)
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

func (s *Store) GetSubcategory(id int64) (models.Subcategory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.d.Subcategories.Items[id]
	if !ok {
		return models.Subcategory{}, notFound("подкатегория не найдена")
	}
	out := *sub
	out.Images = s.mediaFor("subcategory", id)
	return out, nil
}

func (s *Store) CreateSubcategory(in models.SubcategoryInput) (models.Subcategory, error) {
	if in.Name == "" || in.CategoryID == 0 {
		return models.Subcategory{}, badRequest("поля name и category_id обязательны")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Subcategory{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.SubcategoryTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Categories.Items[in.CategoryID]; !ok {
		return models.Subcategory{}, badRequest("указанная категория не существует")
	}
	if s.subcategorySlugTaken(in.Slug, 0) {
		return models.Subcategory{}, conflict("подкатегория с таким slug уже существует")
	}

	now := time.Now().UTC()
	sub := &models.Subcategory{
		ID:         s.d.Subcategories.nextID(),
		CategoryID: in.CategoryID,
		Name:       in.Name,
		Slug:       in.Slug,
		SortOrder:  in.SortOrder,
		Status:     status,
		Language:   in.Language,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.d.Subcategories.Items[sub.ID] = sub
	s.markDirty()

	return *sub, nil
}

func (s *Store) UpdateSubcategory(id int64, in models.SubcategoryInput) (models.Subcategory, error) {
	if in.Name == "" || in.CategoryID == 0 {
		return models.Subcategory{}, badRequest("поля name и category_id обязательны")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Subcategory{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.SubcategoryTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Subcategories.Items[id]
	if !ok {
		return models.Subcategory{}, notFound("подкатегория не найдена")
	}
	if _, ok := s.d.Categories.Items[in.CategoryID]; !ok {
		return models.Subcategory{}, badRequest("указанная категория не существует")
	}
	if s.subcategorySlugTaken(in.Slug, id) {
		return models.Subcategory{}, conflict("подкатегория с таким slug уже существует")
	}

	updated := &models.Subcategory{
		ID:         id,
		CategoryID: in.CategoryID,
		Name:       in.Name,
		Slug:       in.Slug,
		SortOrder:  in.SortOrder,
		Status:     status,
		Language:   in.Language,
		CreatedAt:  old.CreatedAt,
		UpdatedAt:  time.Now().UTC(),
	}
	s.d.Subcategories.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("subcategory", id)
	return out, nil
}

func (s *Store) DeleteSubcategory(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Subcategories.Items[id]; !ok {
		return notFound("подкатегория не найдена")
	}
	// Правило 2: нельзя удалить подкатегорию, пока в ней есть товары.
	for _, p := range s.d.Products.Items {
		if p.SubcategoryID == id {
			return conflict("нельзя удалить подкатегорию: в ней есть товары")
		}
	}

	delete(s.d.Subcategories.Items, id)
	s.deleteMediaForEntity("subcategory", id)
	s.markDirty()
	return nil
}

func (s *Store) subcategorySlugTaken(slug string, exceptID int64) bool {
	for _, sub := range s.d.Subcategories.Items {
		if sub.ID != exceptID && sub.Slug == slug {
			return true
		}
	}
	return false
}
