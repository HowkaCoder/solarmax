package utils

import "encoding/json"

// ToJSONB сериализует значение (обычно map[string]SomeTranslation) в JSON-байты
// для записи в jsonb-колонку. nil-карта превращается в пустой объект "{}",
// а не в JSON null, чтобы в БД всегда лежал валидный объект.
func ToJSONB(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

// FromJSONB разбирает JSON-байты из jsonb-колонки в указанную структуру.
// Пустые/nil байты интерпретируются как пустой объект.
func FromJSONB(data []byte, dest interface{}) error {
	if len(data) == 0 {
		data = []byte("{}")
	}
	return json.Unmarshal(data, dest)
}
