package proxy

import (
	"slices"
	"testing"
)

func TestProxyManager_ParseFallbackContainersFromDockerPS(t *testing.T) {
	t.Parallel()

	dockerPS := "vllm_node\tghcr.io/acme/vllm:latest\napi\tghcr.io/acme/api:latest\nworker\tghcr.io/acme/VLLM-worker:2\nVLLM-extra\tghcr.io/acme/other:1\n"
	got := parseFallbackContainersFromDockerPS(dockerPS, "vllm")
	want := []string{"vllm_node", "worker", "VLLM-extra"}

	if !slices.Equal(got, want) {
		t.Fatalf("parseFallbackContainersFromDockerPS() = %#v, want %#v", got, want)
	}
}

func TestProxyManager_ParseFallbackContainersFromDockerPSEmptyToken(t *testing.T) {
	t.Parallel()

	got := parseFallbackContainersFromDockerPS("vllm_node\timg\n", "")
	if got != nil {
		t.Fatalf("parseFallbackContainersFromDockerPS() with empty token = %#v, want nil", got)
	}
}

func TestProxyManager_NormalizeContainerImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "trimmed", value: "  ghcr.io/acme/vllm:1  ", want: "ghcr.io/acme/vllm:1"},
		{name: "numeric", value: 42, want: "42"},
		{name: "string nil literal", value: "<nil>", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeContainerImage(tc.value)
			if got != tc.want {
				t.Fatalf("normalizeContainerImage(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestProxyManager_CatalogContainerImage(t *testing.T) {
	t.Parallel()

	catalog := map[string]RecipeCatalogItem{
		"/tmp/recipes/exact.yaml": {ContainerImage: "exact:image"},
		"recipe-a":                {ContainerImage: "recipe-a:image"},
	}

	if got := catalogContainerImage(catalog, "/tmp/recipes/exact.yaml"); got != "exact:image" {
		t.Fatalf("catalogContainerImage(exact) = %q, want %q", got, "exact:image")
	}

	if got := catalogContainerImage(catalog, "/opt/backend/recipe-a.yaml"); got != "recipe-a:image" {
		t.Fatalf("catalogContainerImage(basename) = %q, want %q", got, "recipe-a:image")
	}

	if got := catalogContainerImage(catalog, ""); got != "" {
		t.Fatalf("catalogContainerImage(empty) = %q, want empty", got)
	}
}

func TestProxyManager_ResolveModelContainerImagePrecedence(t *testing.T) {
	t.Parallel()

	catalog := map[string]RecipeCatalogItem{
		"catalog-recipe": {ContainerImage: "catalog:recipe"},
		"model-a":        {ContainerImage: "catalog:model"},
	}

	t.Run("recipe metadata container wins", func(t *testing.T) {
		metadata := map[string]any{
			"recipe_ui": map[string]any{
				"container_image": "meta:recipe",
				"recipe_ref":      "catalog-recipe.yaml",
			},
			"container_image": "meta:top",
		}

		got := resolveModelContainerImage("model-a", "", metadata, catalog, "default:image")
		if got != "meta:recipe" {
			t.Fatalf("resolveModelContainerImage(recipe metadata) = %q, want %q", got, "meta:recipe")
		}
	})

	t.Run("top level container image wins when no recipe container", func(t *testing.T) {
		metadata := map[string]any{
			"container_image": "meta:top",
		}

		got := resolveModelContainerImage("model-a", "", metadata, catalog, "default:image")
		if got != "meta:top" {
			t.Fatalf("resolveModelContainerImage(top level) = %q, want %q", got, "meta:top")
		}
	})

	t.Run("catalog by recipe ref", func(t *testing.T) {
		metadata := map[string]any{
			"recipe_ui": map[string]any{
				"recipe_ref": "catalog-recipe.yaml",
			},
		}

		got := resolveModelContainerImage("model-a", "", metadata, catalog, "default:image")
		if got != "catalog:recipe" {
			t.Fatalf("resolveModelContainerImage(catalog recipe) = %q, want %q", got, "catalog:recipe")
		}
	})

	t.Run("catalog by recipe ref from command", func(t *testing.T) {
		got := resolveModelContainerImage("model-a", "${recipe_runner} catalog-recipe", nil, catalog, "default:image")
		if got != "catalog:recipe" {
			t.Fatalf("resolveModelContainerImage(recipe from cmd) = %q, want %q", got, "catalog:recipe")
		}
	})

	t.Run("catalog by model id", func(t *testing.T) {
		got := resolveModelContainerImage("model-a", "", nil, catalog, "default:image")
		if got != "catalog:model" {
			t.Fatalf("resolveModelContainerImage(catalog model) = %q, want %q", got, "catalog:model")
		}
	})

	t.Run("default value fallback", func(t *testing.T) {
		got := resolveModelContainerImage("unknown-model", "", nil, map[string]RecipeCatalogItem{}, "default:image")
		if got != "default:image" {
			t.Fatalf("resolveModelContainerImage(default) = %q, want %q", got, "default:image")
		}
	})

	t.Run("empty when nothing resolves", func(t *testing.T) {
		got := resolveModelContainerImage("unknown-model", "", nil, map[string]RecipeCatalogItem{}, "")
		if got != "" {
			t.Fatalf("resolveModelContainerImage(empty) = %q, want empty", got)
		}
	})
}
