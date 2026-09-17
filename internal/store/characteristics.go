package store

import (
	"sort"
	"time"

	"solarmax/internal/models"
)

func (s *Store) ListCharacteristics() []models.Characteristic {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []models.Characteristic{}
	for _, c := range values(&s.d.Characteristics) {
		out := *c
		out.Values = s.characteristicValues(c.ID)
		list = append(list, out)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Name != list[j].Name {
			return list[i].Name < list[j].Name
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Store) CreateCharacteristic(in models.CharacteristicInput) (models.Characteristic, error) {
	if in.Name == "" {
		return models.Characteristic{}, badRequest("поле name обязательно")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.d.Characteristics.Items {
		if c.Name == in.Name {
			return models.Characteristic{}, conflict("характеристика с таким именем уже существует")
		}
	}

	c := &models.Characteristic{
		ID:        s.d.Characteristics.nextID(),
		Name:      in.Name,
		CreatedAt: time.Now().UTC(),
	}
	s.d.Characteristics.Items[c.ID] = c
	s.markDirty()

	return *c, nil
}

func (s *Store) UpdateCharacteristic(id int64, in models.CharacteristicInput) (models.Characteristic, error) {
	if in.Name == "" {
		return models.Characteristic{}, badRequest("поле name обязательно")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.Characteristics.Items[id]
	if !ok {
		return models.Characteristic{}, notFound("характеристика не найдена")
	}
	for _, c := range s.d.Characteristics.Items {
		if c.ID != id && c.Name == in.Name {
			return models.Characteristic{}, conflict("характеристика с таким именем уже существует")
		}
	}

	updated := &models.Characteristic{
		ID:        id,
		Name:      in.Name,
		CreatedAt: old.CreatedAt,
	}
	s.d.Characteristics.Items[id] = updated
	s.markDirty()

	return *updated, nil
}

func (s *Store) DeleteCharacteristic(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Characteristics.Items[id]; !ok {
		return notFound("характеристика не найдена")
	}

	delete(s.d.Characteristics.Items, id)
	// Каскад: значения характеристики и все связи с товарами.
	for vID, v := range s.d.CharValues.Items {
		if v.CharacteristicID == id {
			delete(s.d.CharValues.Items, vID)
		}
	}
	for pcID, pc := range s.d.ProductChars.Items {
		if pc.CharacteristicID == id {
			delete(s.d.ProductChars.Items, pcID)
		}
	}
	s.markDirty()
	return nil
}

func (s *Store) ListCharacteristicValues(characteristicID int64) []models.CharacteristicValue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := s.characteristicValues(characteristicID)
	if list == nil {
		return []models.CharacteristicValue{}
	}
	return list
}

// characteristicValues возвращает значения характеристики.
// Вызывается под уже взятым локом.
func (s *Store) characteristicValues(characteristicID int64) []models.CharacteristicValue {
	var list []models.CharacteristicValue
	for _, v := range s.d.CharValues.Items {
		if v.CharacteristicID == characteristicID {
			list = append(list, *v)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Value != list[j].Value {
			return list[i].Value < list[j].Value
		}
		return list[i].ID < list[j].ID
	})
	return list
}

func (s *Store) CreateCharacteristicValue(characteristicID int64, in models.CharacteristicValueInput) (models.CharacteristicValue, error) {
	if in.Value == "" {
		return models.CharacteristicValue{}, badRequest("поле value обязательно")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Characteristics.Items[characteristicID]; !ok {
		return models.CharacteristicValue{}, badRequest("указанная характеристика не существует")
	}
	for _, v := range s.d.CharValues.Items {
		if v.CharacteristicID == characteristicID && v.Value == in.Value {
			return models.CharacteristicValue{}, conflict("такое значение уже существует для этой характеристики")
		}
	}

	v := &models.CharacteristicValue{
		ID:               s.d.CharValues.nextID(),
		CharacteristicID: characteristicID,
		Value:            in.Value,
		CreatedAt:        time.Now().UTC(),
	}
	s.d.CharValues.Items[v.ID] = v
	s.markDirty()

	return *v, nil
}

func (s *Store) UpdateCharacteristicValue(valueID int64, in models.CharacteristicValueInput) (models.CharacteristicValue, error) {
	if in.Value == "" {
		return models.CharacteristicValue{}, badRequest("поле value обязательно")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.d.CharValues.Items[valueID]
	if !ok {
		return models.CharacteristicValue{}, notFound("значение не найдено")
	}
	for _, v := range s.d.CharValues.Items {
		if v.ID != valueID && v.CharacteristicID == old.CharacteristicID && v.Value == in.Value {
			return models.CharacteristicValue{}, conflict("такое значение уже существует для этой характеристики")
		}
	}

	updated := &models.CharacteristicValue{
		ID:               valueID,
		CharacteristicID: old.CharacteristicID,
		Value:            in.Value,
		CreatedAt:        old.CreatedAt,
	}
	s.d.CharValues.Items[valueID] = updated
	s.markDirty()

	return *updated, nil
}

func (s *Store) DeleteCharacteristicValue(valueID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.CharValues.Items[valueID]; !ok {
		return notFound("значение не найдено")
	}

	delete(s.d.CharValues.Items, valueID)
	for pcID, pc := range s.d.ProductChars.Items {
		if pc.ValueID == valueID {
			delete(s.d.ProductChars.Items, pcID)
		}
	}
	s.markDirty()
	return nil
}

// ==================== ХАРАКТЕРИСТИКИ ТОВАРА ====================

func (s *Store) GetProductCharacteristics(productID int64) []models.ProductCharacteristic {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := s.productCharacteristics(productID)
	if list == nil {
		return []models.ProductCharacteristic{}
	}
	return list
}

// productCharacteristics собирает связи товара вместе с именами
// характеристик и значений (аналог JOIN-ов в прежнем SQL-запросе).
// Вызывается под уже взятым локом.
func (s *Store) productCharacteristics(productID int64) []models.ProductCharacteristic {
	var list []models.ProductCharacteristic
	for _, pc := range s.d.ProductChars.Items {
		if pc.ProductID != productID {
			continue
		}
		out := *pc
		if c, ok := s.d.Characteristics.Items[pc.CharacteristicID]; ok {
			out.CharacteristicName = c.Name
		}
		if v, ok := s.d.CharValues.Items[pc.ValueID]; ok {
			out.Value = v.Value
		}
		list = append(list, out)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].CharacteristicName != list[j].CharacteristicName {
			return list[i].CharacteristicName < list[j].CharacteristicName
		}
		return list[i].ID < list[j].ID
	})
	return list
}

// SetProductCharacteristic создаёт или перезаписывает значение характеристики
// у товара (аналог INSERT ... ON CONFLICT DO UPDATE).
func (s *Store) SetProductCharacteristic(productID int64, in models.ProductCharacteristicInput) ([]models.ProductCharacteristic, error) {
	if in.CharacteristicID == 0 || in.ValueID == 0 {
		return nil, badRequest("поля characteristic_id и value_id обязательны")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.d.Products.Items[productID]; !ok {
		return nil, badRequest("указанный товар, характеристика или значение не существуют")
	}
	if _, ok := s.d.Characteristics.Items[in.CharacteristicID]; !ok {
		return nil, badRequest("указанный товар, характеристика или значение не существуют")
	}
	value, ok := s.d.CharValues.Items[in.ValueID]
	if !ok {
		return nil, badRequest("указанный товар, характеристика или значение не существуют")
	}
	if value.CharacteristicID != in.CharacteristicID {
		return nil, badRequest("значение принадлежит другой характеристике")
	}

	replaced := false
	for id, pc := range s.d.ProductChars.Items {
		if pc.ProductID == productID && pc.CharacteristicID == in.CharacteristicID {
			updated := *pc
			updated.ValueID = in.ValueID
			s.d.ProductChars.Items[id] = &updated
			replaced = true
			break
		}
	}
	if !replaced {
		pc := &models.ProductCharacteristic{
			ID:               s.d.ProductChars.nextID(),
			ProductID:        productID,
			CharacteristicID: in.CharacteristicID,
			ValueID:          in.ValueID,
		}
		s.d.ProductChars.Items[pc.ID] = pc
	}
	s.markDirty()

	list := s.productCharacteristics(productID)
	if list == nil {
		list = []models.ProductCharacteristic{}
	}
	return list, nil
}

func (s *Store) DeleteProductCharacteristic(productID, characteristicID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, pc := range s.d.ProductChars.Items {
		if pc.ProductID == productID && pc.CharacteristicID == characteristicID {
			delete(s.d.ProductChars.Items, id)
			s.markDirty()
			return nil
		}
	}
	return notFound("связь не найдена")
}
