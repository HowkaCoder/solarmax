// Package store - встроенное хранилище данных вместо внешней базы.
//
// Все данные живут в памяти процесса (обычные map под RWMutex) и по желанию
// сбрасываются в один JSON-файл на диске. Никакого Postgres, никакого CGO,
// никаких внешних зависимостей: приложение - один бинарник, который можно
// запустить где угодно, в том числе на бесплатном хостинге.
//
// Важный инвариант: записи в хранилище неизменяемы. Обновление всегда кладёт
// в map новый объект вместо старого, а наружу отдаётся копия. Благодаря этому
// читатели никогда не видят объект в промежуточном состоянии.
package store

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"solarmax/internal/models"
)

// ==================== ОШИБКИ ====================

type Kind int

const (
	KindBadRequest Kind = iota
	KindNotFound
	KindConflict
)

// Error - ошибка хранилища с готовым человекочитаемым текстом,
// который хендлеры отдают клиенту как есть.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func badRequest(msg string) error { return &Error{Kind: KindBadRequest, Msg: msg} }
func notFound(msg string) error   { return &Error{Kind: KindNotFound, Msg: msg} }
func conflict(msg string) error   { return &Error{Kind: KindConflict, Msg: msg} }

// HTTPStatus переводит ошибку хранилища в HTTP-код.
func HTTPStatus(err error) int {
	var e *Error
	if ok := asStoreError(err, &e); ok {
		switch e.Kind {
		case KindNotFound:
			return http.StatusNotFound
		case KindConflict:
			return http.StatusConflict
		default:
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}

func asStoreError(err error, target **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if ok {
		*target = e
	}
	return ok
}

// ==================== ТАБЛИЦЫ ====================

// table - аналог таблицы: автоинкремент + записи по id.
type table[T any] struct {
	Seq   int64        `json:"seq"`
	Items map[int64]*T `json:"items"`
}

func (t *table[T]) init() {
	if t.Items == nil {
		t.Items = map[int64]*T{}
	}
}

func (t *table[T]) nextID() int64 {
	t.Seq++
	return t.Seq
}

// storedAdmin лежит только внутри хранилища: хэш пароля наружу не уезжает.
type storedAdmin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// data - полный снапшот хранилища, ровно он и пишется в JSON-файл.
type data struct {
	Version         int                                 `json:"version"`
	SavedAt         time.Time                           `json:"saved_at"`
	Categories      table[models.Category]              `json:"categories"`
	Subcategories   table[models.Subcategory]           `json:"subcategories"`
	Products        table[models.Product]               `json:"products"`
	Services        table[models.Service]               `json:"services"`
	ObjectTypes     table[models.ObjectType]            `json:"object_types"`
	Advantages      table[models.Advantage]             `json:"advantages"`
	Characteristics table[models.Characteristic]        `json:"characteristics"`
	CharValues      table[models.CharacteristicValue]   `json:"characteristic_values"`
	ProductChars    table[models.ProductCharacteristic] `json:"product_characteristics"`
	Media           table[models.Media]                 `json:"media"`
	Admins          table[storedAdmin]                  `json:"admins"`
}

func (d *data) init() {
	d.Version = 1
	d.Categories.init()
	d.Subcategories.init()
	d.Products.init()
	d.Services.init()
	d.ObjectTypes.init()
	d.Advantages.init()
	d.Characteristics.init()
	d.CharValues.init()
	d.ProductChars.init()
	d.Media.init()
	d.Admins.init()
}

// ==================== ХРАНИЛИЩЕ ====================

type Store struct {
	mu      sync.RWMutex
	d       *data
	path    string
	persist bool

	dirtyMu sync.Mutex
	dirty   bool
}

// New создаёт хранилище. Если persist=true и файл снапшота существует,
// данные поднимаются из него; иначе стартуем с пустого состояния.
func New(path string, persist bool) (*Store, error) {
	s := &Store{
		d:       &data{},
		path:    path,
		persist: persist,
	}
	s.d.init()

	if !persist {
		return s, nil
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("не удалось создать папку данных: %w", err)
		}
	}

	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл данных: %w", err)
	}
	if len(raw) == 0 {
		return s, nil
	}

	var loaded data
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, fmt.Errorf("файл данных повреждён (%s): %w", path, err)
	}
	loaded.init()
	s.d = &loaded

	return s, nil
}

// FilePath - путь к файлу снапшота (для логов и /health).
func (s *Store) FilePath() string { return s.path }

// Persistent сообщает, включено ли сохранение на диск.
func (s *Store) Persistent() bool { return s.persist }

// markDirty помечает, что состояние изменилось и его нужно сохранить.
// Вызывается под write-локом.
func (s *Store) markDirty() {
	s.dirtyMu.Lock()
	s.dirty = true
	s.dirtyMu.Unlock()
}

func (s *Store) takeDirty() bool {
	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()
	was := s.dirty
	s.dirty = false
	return was
}

// Save атомарно записывает снапшот на диск (временный файл + rename),
// чтобы даже при падении процесса на диске остался целый файл.
func (s *Store) Save() error {
	if !s.persist {
		return nil
	}

	s.mu.RLock()
	s.d.SavedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(s.d, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// StartAutosave раз в interval сбрасывает изменения на диск в фоне
// и делает финальное сохранение при остановке (done закрыт).
func (s *Store) StartAutosave(done <-chan struct{}, interval time.Duration) {
	if !s.persist {
		return
	}
	if interval <= 0 {
		interval = 3 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if s.takeDirty() {
					if err := s.Save(); err != nil {
						fmt.Printf("не удалось сохранить данные: %v\n", err)
					}
				}
			case <-done:
				if err := s.Save(); err != nil {
					fmt.Printf("не удалось сохранить данные при остановке: %v\n", err)
				}
				return
			}
		}
	}()
}

// Stats - короткая сводка по количеству записей (для /health).
type Stats struct {
	Categories      int `json:"categories"`
	Subcategories   int `json:"subcategories"`
	Products        int `json:"products"`
	Services        int `json:"services"`
	ObjectTypes     int `json:"object_types"`
	Advantages      int `json:"advantages"`
	Characteristics int `json:"characteristics"`
	Media           int `json:"media"`
	Admins          int `json:"admins"`
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Stats{
		Categories:      len(s.d.Categories.Items),
		Subcategories:   len(s.d.Subcategories.Items),
		Products:        len(s.d.Products.Items),
		Services:        len(s.d.Services.Items),
		ObjectTypes:     len(s.d.ObjectTypes.Items),
		Advantages:      len(s.d.Advantages.Items),
		Characteristics: len(s.d.Characteristics.Items),
		Media:           len(s.d.Media.Items),
		Admins:          len(s.d.Admins.Items),
	}
}

// ==================== ОБЩИЕ ХЕЛПЕРЫ ====================

// values возвращает все записи таблицы в виде среза (порядок неопределён,
// поэтому вызывающий код обязан сам отсортировать результат).
func values[T any](t *table[T]) []*T {
	out := make([]*T, 0, len(t.Items))
	for _, v := range t.Items {
		out = append(out, v)
	}
	return out
}

// hasLang проверяет, есть ли перевод на указанный язык
// (аналог jsonb-оператора `translations ? 'ru'` в Postgres).
func hasLang[T any](m map[string]T, code string) bool {
	if code == "" {
		return true
	}
	_, ok := m[code]
	return ok
}

func normalizeStatus(status string) (string, error) {
	switch status {
	case "":
		return "active", nil
	case "active", "inactive":
		return status, nil
	default:
		return "", badRequest("status должен быть active или inactive")
	}
}
