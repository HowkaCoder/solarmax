package store

import (
	"net/http"
	"path/filepath"
	"testing"

	"solarmax/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "data.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCatalog(t *testing.T) {
	s := newTestStore(t)

	cat, err := s.CreateCategory(models.CategoryInput{Name: "Solar panels"})
	if err != nil {
		t.Fatal(err)
	}
	if cat.Slug == "" {
		t.Fatal("slug was not generated")
	}

	if _, err := s.CreateCategory(models.CategoryInput{Name: "Solar panels"}); HTTPStatus(err) != http.StatusConflict {
		t.Fatalf("duplicate slug: expected 409, got %v", err)
	}

	sub, err := s.CreateSubcategory(models.SubcategoryInput{CategoryID: cat.ID, Name: "Mono"})
	if err != nil {
		t.Fatal(err)
	}

	price := 120.5
	p, err := s.CreateProduct(models.ProductInput{SubcategoryID: sub.ID, Name: "Panel 450W", Price: &price})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetProduct(p.ID); got.Name != "Panel 450W" {
		t.Fatalf("product not stored: %+v", got)
	}

	if err := s.DeleteCategory(cat.ID); HTTPStatus(err) != http.StatusConflict {
		t.Fatalf("category with subcategories must not be deleted, got %v", err)
	}
	if _, err := s.CreateProduct(models.ProductInput{SubcategoryID: 999, Name: "x"}); HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("unknown subcategory: expected 400, got %v", err)
	}
	if _, err := s.GetProduct(999); HTTPStatus(err) != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	s, err := New(path, true)
	if err != nil {
		t.Fatal(err)
	}
	cat, err := s.CreateCategory(models.CategoryInput{Name: "Inverters"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := New(path, true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reloaded.GetCategory(cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Inverters" {
		t.Fatalf("name = %q", got.Name)
	}

	next, err := reloaded.CreateCategory(models.CategoryInput{Name: "Batteries"})
	if err != nil {
		t.Fatal(err)
	}
	if next.ID <= cat.ID {
		t.Fatalf("ids are reused after reload: %d <= %d", next.ID, cat.ID)
	}
}

func TestSnapshotRestore(t *testing.T) {
	a := newTestStore(t)
	if _, err := a.CreateCategory(models.CategoryInput{Name: "Cables"}); err != nil {
		t.Fatal(err)
	}
	raw, err := a.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	b := newTestStore(t)
	if err := b.Restore(raw); err != nil {
		t.Fatal(err)
	}
	if n := len(b.ListCategories("", "")); n != 1 {
		t.Fatalf("restored %d categories", n)
	}
}
