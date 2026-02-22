package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectRecipeBackendKind(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/home/u/spark-vllm-docker", want: "vllm"},
		{path: "/home/u/sqlang-backend", want: "sqlang"},
		{path: "/home/u/trtllm-backend", want: "trtllm"},
		{path: "/home/u/spark-llama-cpp", want: "llamacpp"},
		{path: "/opt/custom-backend", want: "custom"},
	}

	for _, tc := range tests {
		got := detectRecipeBackendKind(tc.path)
		if got != tc.want {
			t.Fatalf("detectRecipeBackendKind(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestLatestTRTLLMTag(t *testing.T) {
	tags := []string{"1.3.0rc3", "1.2.5", "1.3.0", "1.3.0rc4", "1.4.0rc1", "latest", "1.4.0"}
	got := latestTRTLLMTag(tags)
	want := "1.4.0"
	if got != want {
		t.Fatalf("latestTRTLLMTag() = %q, want %q", got, want)
	}
}

func TestCompareTRTLLMTagVersion(t *testing.T) {
	a, ok := parseTRTLLMTagVersion("1.3.0rc3")
	if !ok {
		t.Fatalf("failed to parse a")
	}
	b, ok := parseTRTLLMTagVersion("1.3.0")
	if !ok {
		t.Fatalf("failed to parse b")
	}
	if compareTRTLLMTagVersion(a, b) >= 0 {
		t.Fatalf("expected rc version to be lower than stable")
	}

	c, ok := parseTRTLLMTagVersion("1.3.1")
	if !ok {
		t.Fatalf("failed to parse c")
	}
	if compareTRTLLMTagVersion(b, c) >= 0 {
		t.Fatalf("expected 1.3.0 < 1.3.1")
	}
}

func TestResolveTRTLLMSourceImagePrefersOverrideFile(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, trtllmSourceImageOverrideFile)
	overrideValue := "nvcr.io/nvidia/tensorrt-llm/release:1.4.0"
	if err := os.WriteFile(overridePath, []byte(overrideValue+"\n"), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	got := resolveTRTLLMSourceImage(dir, "")
	if got != overrideValue {
		t.Fatalf("resolveTRTLLMSourceImage() = %q, want %q", got, overrideValue)
	}
}

func TestResolveLLAMACPPSourceImagePrefersOverrideFile(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, llamacppSourceImageOverrideFile)
	overrideValue := "llama-cpp-spark:custom"
	if err := os.WriteFile(overridePath, []byte(overrideValue+"\n"), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	got := resolveLLAMACPPSourceImage(dir, "")
	if got != overrideValue {
		t.Fatalf("resolveLLAMACPPSourceImage() = %q, want %q", got, overrideValue)
	}
}

func TestBackendScopedConfigPath(t *testing.T) {
	cfg := "/tmp/config.yaml"
	if got, want := backendScopedConfigPath(cfg, "vllm"), "/tmp/config.vllm.yaml"; got != want {
		t.Fatalf("backendScopedConfigPath(vllm) = %q, want %q", got, want)
	}
	if got, want := backendScopedConfigPath(cfg, "trtllm"), "/tmp/config.trtllm.yaml"; got != want {
		t.Fatalf("backendScopedConfigPath(trtllm) = %q, want %q", got, want)
	}
	if got, want := backendScopedConfigPath(cfg, "unknown"), "/tmp/config.custom.yaml"; got != want {
		t.Fatalf("backendScopedConfigPath(custom) = %q, want %q", got, want)
	}
}

func TestSwitchRecipeBackendConfigPersistsAndRestores(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "config.yaml")
	vllmBody := `macros:
  recipe_runner: /vllm/run-recipe.sh
`
	trtBody := `macros:
  recipe_runner: /trt/run-recipe.sh
`

	if err := os.WriteFile(activePath, []byte(vllmBody), 0o644); err != nil {
		t.Fatalf("write active config: %v", err)
	}
	trtPath := backendScopedConfigPath(activePath, "trtllm")
	if err := os.WriteFile(trtPath, []byte(trtBody), 0o644); err != nil {
		t.Fatalf("write trt config: %v", err)
	}

	pm := &ProxyManager{configPath: activePath}
	if err := pm.switchRecipeBackendConfig("vllm", "trtllm"); err != nil {
		t.Fatalf("switchRecipeBackendConfig error: %v", err)
	}

	activeGot, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read active config: %v", err)
	}
	if string(activeGot) != trtBody {
		t.Fatalf("active config mismatch\nwant:\n%s\ngot:\n%s", trtBody, string(activeGot))
	}

	vllmPath := backendScopedConfigPath(activePath, "vllm")
	vllmGot, err := os.ReadFile(vllmPath)
	if err != nil {
		t.Fatalf("read vllm scoped config: %v", err)
	}
	if string(vllmGot) != vllmBody {
		t.Fatalf("vllm scoped config mismatch\nwant:\n%s\ngot:\n%s", vllmBody, string(vllmGot))
	}
}

func TestRecipeManagedModelInCatalog(t *testing.T) {
	catalog := map[string]RecipeCatalogItem{
		"qwen3-coder-next-vllm-next": {ID: "qwen3-coder-next-vllm-next"},
	}

	if !recipeManagedModelInCatalog(RecipeManagedModel{}, catalog) {
		t.Fatalf("empty recipeRef should be allowed")
	}
	if !recipeManagedModelInCatalog(RecipeManagedModel{RecipeRef: "qwen3-coder-next-vllm-next"}, catalog) {
		t.Fatalf("known recipeRef should be allowed")
	}
	if recipeManagedModelInCatalog(RecipeManagedModel{RecipeRef: "openai-gpt-oss-120b"}, catalog) {
		t.Fatalf("unknown recipeRef should be filtered out")
	}
}

func TestRecipeEntryTargetsActiveBackend(t *testing.T) {
	active := filepath.Join(t.TempDir(), "backend-active")
	other := filepath.Join(t.TempDir(), "backend-other")

	if !recipeEntryTargetsActiveBackend(nil, active) {
		t.Fatalf("nil metadata should be allowed")
	}
	if !recipeEntryTargetsActiveBackend(map[string]any{}, active) {
		t.Fatalf("missing recipe metadata should be allowed")
	}

	metaActive := map[string]any{recipeMetadataKey: map[string]any{"backend_dir": active}}
	if !recipeEntryTargetsActiveBackend(metaActive, active) {
		t.Fatalf("matching backend_dir should be allowed")
	}

	metaOther := map[string]any{recipeMetadataKey: map[string]any{"backend_dir": other}}
	if recipeEntryTargetsActiveBackend(metaOther, active) {
		t.Fatalf("different backend_dir should be filtered out")
	}
}

func TestResolveHFDownloadScriptPathPrefersEnv(t *testing.T) {
	temp := t.TempDir()
	script := filepath.Join(temp, "hf-download.sh")
	t.Setenv(hfDownloadScriptPathEnv, script)

	if got := resolveHFDownloadScriptPath(); got != script {
		t.Fatalf("resolveHFDownloadScriptPath() = %q, want %q", got, script)
	}
}

func TestRecipeBackendActionsForKindIncludesHFDownload(t *testing.T) {
	temp := t.TempDir()
	script := filepath.Join(temp, "hf-download.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	t.Setenv(hfDownloadScriptPathEnv, script)

	actions := recipeBackendActionsForKind("vllm", temp, "")
	for _, action := range actions {
		if action.Action == "download_hf_model" {
			if !strings.Contains(action.CommandHint, script) {
				t.Fatalf("download_hf_model commandHint missing script path: %q", action.CommandHint)
			}
			return
		}
	}
	t.Fatalf("download_hf_model action not found")
}

func TestResolveLLAMACPPHFDownloadScriptPathPrefersEnv(t *testing.T) {
	temp := t.TempDir()
	script := filepath.Join(temp, "llamacpp-hf-download.sh")
	t.Setenv(llamacppHFDownloadScriptPathEnv, script)

	if got := resolveLLAMACPPHFDownloadScriptPath(); got != script {
		t.Fatalf("resolveLLAMACPPHFDownloadScriptPath() = %q, want %q", got, script)
	}
}

func TestRecipeBackendActionsForKindIncludesLLAMACPPQuickDownload(t *testing.T) {
	temp := t.TempDir()
	hfScript := filepath.Join(temp, "hf-download.sh")
	if err := os.WriteFile(hfScript, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write hf script: %v", err)
	}
	llamaScript := filepath.Join(temp, "llamacpp-hf-download.sh")
	if err := os.WriteFile(llamaScript, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write llamacpp script: %v", err)
	}
	t.Setenv(hfDownloadScriptPathEnv, hfScript)
	t.Setenv(llamacppHFDownloadScriptPathEnv, llamaScript)

	actions := recipeBackendActionsForKind("llamacpp", temp, "")
	for _, action := range actions {
		if action.Action == "download_llamacpp_q8_model" {
			if !strings.Contains(action.CommandHint, llamaScript) {
				t.Fatalf("download_llamacpp_q8_model commandHint missing script path: %q", action.CommandHint)
			}
			if !strings.Contains(action.CommandHint, defaultLLAMACPPHFModel) {
				t.Fatalf("download_llamacpp_q8_model commandHint missing model: %q", action.CommandHint)
			}
			if !strings.Contains(action.CommandHint, defaultLLAMACPPHFIncludePattern) {
				t.Fatalf("download_llamacpp_q8_model commandHint missing include pattern: %q", action.CommandHint)
			}
			return
		}
	}
	t.Fatalf("download_llamacpp_q8_model action not found")
}
