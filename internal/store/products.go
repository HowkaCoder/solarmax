package store

import (
	"sort"
	"time"

	"solarmax/internal/models"
	"solarmax/internal/utils"
)

func (s *Store) ListProducts(status, language string, subcategoryID int64) []models.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Product{}
	for _, p := range values(&s.d.Products) {
		if status != "" && p.Status != status {
			continue
		}
		if subcategoryID > 0 && p.SubcategoryID != subcategoryID {
			continue
		}
		if !hasLang(p.Language, language) {
			continue
		}
		out := *p
		out.Images = s.mediaFor("product", p.ID)
		list = append(list, out)
	}

	// Как и раньше: свежие товары сверху.
	sort.Slice(list, func(i, j int) bool { return list[i].ID > list[j].ID })
	return list
}

func (s *Store) GetProduct(id int64) (models.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.d.Products.Items[id]
	if !ok {
		return models.Product{}, notFound("товар не найден")
	}
	out := *p
	out.Images = s.mediaFor("product", id)
	out.Characteristics = s.productCharacteristics(id)
	return out, nil
}

func (s *Store) CreateProduct(in models.ProductInput) (models.Product, error) {
	if in.Name == "" || in.SubcategoryID == 0 {
		return models.Product{}, badRequest("поля name и subcategory_id обязательны")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Product{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ProductTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Subcategories.Items[in.SubcategoryID]; !ok {
		return models.Product{}, badRequest("указанная подкатегория не существует")
	}
	if s.productSlugTaken(in.Slug, 0) {
		return models.Product{}, conflict("товар с таким slug уже существует")
	}

	now := time.Now().UTC()
	p := &models.Product{
		ID:            s.d.Products.nextID(),
		SubcategoryID: in.SubcategoryID,
		Name:          in.Name,
		Price:         in.Price,
		Slug:          in.Slug,
		Status:        status,
		Language:      in.Language,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.d.Products.Items[p.ID] = p
	s.markDirty()

	return *p, nil
}

func (s *Store) UpdateProduct(id int64, in models.ProductInput) (models.Product, error) {
	if in.Name == "" || in.SubcategoryID == 0 {
		return models.Product{}, badRequest("поля name и subcategory_id обязательны")
	}
	status, err := normalizeStatus(in.Status)
	if err != nil {
		return models.Product{}, err
	}
	if in.Slug == "" {
		in.Slug = utils.Slugify(in.Name)
	}
	if in.Language == nil {
		in.Language = map[string]models.ProductTranslation{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Products.Items[id]
	if !ok {
		return models.Product{}, notFound("товар не найден")
	}
	if _, ok := s.d.Subcategories.Items[in.SubcategoryID]; !ok {
		return models.Product{}, badRequest("указанная подкатегория не существует")
	}
	if s.productSlugTaken(in.Slug, id) {
		return models.Product{}, conflict("товар с таким slug уже существует")
	}

	updated := &models.Product{
		ID:            id,
		SubcategoryID: in.SubcategoryID,
		Name:          in.Name,
		Price:         in.Price,
		Slug:          in.Slug,
		Status:        status,
		Language:      in.Language,
		CreatedAt:     old.CreatedAt,
		UpdatedAt:     time.Now().UTC(),
	}
	s.d.Products.Items[id] = updated
	s.markDirty()

	out := *updated
	out.Images = s.mediaFor("product", id)
	out.Characteristics = s.productCharacteristics(id)
	return out, nil
}

// SetProductStatus - Правило 3: товар можно скрыть, не удаляя.
func (s *Store) SetProductStatus(id int64, status string) error {
	if status != "active" && status != "inactive" {
		return badRequest("status должен быть active или inactive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Products.Items[id]
	if !ok {
		return notFound("товар не найден")
	}

	updated := *old
	updated.Status = status
	updated.UpdatedAt = time.Now().UTC()
	s.d.Products.Items[id] = &updated
	s.markDirty()
	return nil
}

func (s *Store) DeleteProduct(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Products.Items[id]; !ok {
		return notFound("товар не найден")
	}

	delete(s.d.Products.Items, id)
	// Каскад: связи товара с характеристиками и его картинки.
	for pcID, pc := range s.d.ProductChars.Items {
		if pc.ProductID == id {
			delete(s.d.ProductChars.Items, pcID)
		}
	}
	s.deleteMediaForEntity("product", id)
	s.markDirty()
	return nil
}

func (s *Store) productSlugTaken(slug string, exceptID int64) bool {
	for _, p := range s.d.Products.Items {
		if p.ID != exceptID && p.Slug == slug {
			return true
		}
	}
	return false
}
