<script lang="ts">
  import { persistentStore } from "../stores/persistent";
  import { getDockerContainers, getSelectedContainer, setSelectedContainer } from "../stores/api";
  import { onMount } from "svelte";

  // Container definitions
  const containerInfo: Record<string, { name: string; description: string }> = {
    "vllm-next:latest": { name: "vLLM Next", description: "Latest vLLM Next with optimizations" },
    "vllm-node-12.0f:latest": { name: "vLLM Node 12.0f", description: "vLLM Node 12.0f stable" },
    "vllm-node:latest": { name: "vLLM Node", description: "Standard vLLM Node" },
    "vllm-node-mxfp4:latest": { name: "vLLM Node MXFP4", description: "vLLM Node with MXFP4 support" },
  };

  // Persistent store for selected container (fallback)
  const selectedContainerStore = persistentStore<string>("selectedVllmContainer", "vllm-node:latest");

  let isOpen = $state(false);
  let selectedId = $state("vllm-node:latest");
  let availableContainers = $state<string[]>([]);
  let isLoading = $state(false);

  // Load containers and selected container on mount
  onMount(async () => {
    await loadContainers();
    await loadSelectedContainer();
  });

  async function loadContainers(): Promise<void> {
    isLoading = true;
    try {
      const containers = await getDockerContainers();
      availableContainers = containers;
    } catch (error) {
      console.error("Failed to load containers:", error);
      // Fallback to default containers
      availableContainers = ["vllm-next:latest", "vllm-node:latest", "vllm-node-12.0f:latest", "vllm-node-mxfp4:latest"];
    } finally {
      isLoading = false;
    }
  }

  async function loadSelectedContainer(): Promise<void> {
    try {
      const container = await getSelectedContainer();
      selectedId = container;
    } catch (error) {
      console.error("Failed to load selected container:", error);
      // Fallback to persistent store
      selectedId = $selectedContainerStore;
    }
  }

  async function selectContainer(containerId: string): Promise<void> {
    try {
      isLoading = true;
      await setSelectedContainer(containerId);
      selectedId = containerId;
      selectedContainerStore.set(containerId);
      isOpen = false;
    } catch (error) {
      console.error("Failed to set container:", error);
      alert(`Error al seleccionar contenedor: ${error instanceof Error ? error.message : String(error)}`);
    } finally {
      isLoading = false;
    }
  }

  function getSelectedContainerInfo() {
    return containerInfo[selectedId] || containerInfo["vllm-node:latest"];
  }

  function getContainerList() {
    return availableContainers.map(id => ({
      id,
      ...containerInfo[id],
      name: containerInfo[id]?.name || id,
      description: containerInfo[id]?.description || `Container: ${id}`
    }));
  }

  // Export functions for parent components
  export function getSelectedContainerId(): string {
    return selectedId;
  }
</script>

<div class="container-selector">
  <div class="dropdown">
    <button
      class="dropdown-button btn"
      onclick={() => (isOpen = !isOpen)}
      aria-label="Select vLLM container"
      aria-expanded={isOpen}
    >
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-5 h-5">
        <path fill-rule="evenodd" d="M4.5 3.75a3 3 0 0 0-3 3v10.5a3 3 0 0 0 3 3h15a3 3 0 0 0 3-3V6.75a3 3 0 0 0-3-3h-15Zm4.125 5.25a.75.75 0 0 1 .75-.75h9.75a.75.75 0 0 1 0 1.5h-9.75a.75.75 0 0 1-.75-.75Zm.75 2.625a.75.75 0 0 0 0 1.5h9.75a.75.75 0 0 0 0-1.5h-9.75Zm0 2.625a.75.75 0 0 0 0 1.5h5.25a.75.75 0 0 0 0-1.5h-5.25Z" clip-rule="evenodd" />
      </svg>
      <span class="container-name">{getSelectedContainerInfo().name}</span>
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-4 h-4 dropdown-icon" class:rotate={isOpen}>
        <path fill-rule="evenodd" d="M12.53 16.28a.75.75 0 0 1-1.06 0l-7.5-7.5a.75.75 0 0 1 1.06-1.06L12 14.69l6.97-6.97a.75.75 0 1 1 1.06 1.06l-7.5 7.5Z" clip-rule="evenodd" />
      </svg>
    </button>

    {#if isOpen}
      <div class="dropdown-menu">
        <div class="dropdown-header">
          <h3>Select vLLM Container</h3>
          <p class="dropdown-description">Choose the Docker container for running models</p>
        </div>
        <div class="dropdown-items" role="menu">
          {#if isLoading}
            <div class="dropdown-item">
              <div class="item-info">
                <div class="item-name">Cargando contenedores...</div>
              </div>
            </div>
          {:else}
            {#each getContainerList() as container (container.id)}
              <div
                class="dropdown-item"
                class:selected={selectedId === container.id}
                role="menuitem"
                onclick={() => selectContainer(container.id)}
                onkeydown={(e) => e.key === 'Enter' && selectContainer(container.id)}
                tabindex="0"
              >
                <div class="item-info">
                  <div class="item-name">{container.name}</div>
                  <div class="item-description">{container.description}</div>
                  <div class="item-id">{container.id}</div>
                </div>
                {#if selectedId === container.id}
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-5 h-5 check-icon">
                    <path fill-rule="evenodd" d="M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353 8.493-12.739a.75.75 0 0 1 1.04-.208Z" clip-rule="evenodd" />
                  </svg>
                {/if}
              </div>
            {/each}
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .container-selector {
    position: relative;
    display: inline-block;
  }

  .dropdown {
    position: relative;
  }

  .dropdown-button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    background: var(--surface-color, #ffffff);
    border: 1px solid var(--border-color, #e5e7eb);
    border-radius: 0.375rem;
    color: var(--text-color, #1f2937);
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
  }

  .dropdown-button:hover {
    background: var(--surface-hover-color, #f3f4f6);
    border-color: var(--border-hover-color, #d1d5db);
  }

  .container-name {
    font-weight: 500;
    font-size: 0.875rem;
  }

  .dropdown-icon {
    transition: transform 0.2s;
    flex-shrink: 0;
  }

  .dropdown-icon.rotate {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    top: calc(100% + 0.5rem);
    right: 0;
    z-index: 50;
    width: 20rem;
    background: var(--surface-color, #ffffff);
    border: 1px solid var(--border-color, #e5e7eb);
    border-radius: 0.5rem;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
    overflow: hidden;
  }

  .dropdown-header {
    padding: 1rem;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
    background: var(--surface-secondary-color, #f9fafb);
  }

  .dropdown-header h3 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-color, #1f2937);
  }

  .dropdown-description {
    margin: 0.25rem 0 0 0;
    font-size: 0.75rem;
    color: var(--text-secondary-color, #6b7280);
  }

  .dropdown-items {
    max-height: 20rem;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem;
    border-radius: 0.375rem;
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid transparent;
  }

  .dropdown-item:hover {
    background: var(--surface-hover-color, #f3f4f6);
    border-color: var(--border-hover-color, #d1d5db);
  }

  .dropdown-item.selected {
    background: var(--primary-bg-color, #eff6ff);
    border-color: var(--primary-color, #3b82f6);
  }

  .item-info {
    flex: 1;
    min-width: 0;
  }

  .item-name {
    font-weight: 500;
    font-size: 0.875rem;
    color: var(--text-color, #1f2937);
    margin-bottom: 0.125rem;
  }

  .item-description {
    font-size: 0.75rem;
    color: var(--text-secondary-color, #6b7280);
    margin-bottom: 0.125rem;
  }

  .item-id {
    font-size: 0.625rem;
    color: var(--text-tertiary-color, #9ca3af);
    font-family: monospace;
  }

  .check-icon {
    color: var(--primary-color, #3b82f6);
    flex-shrink: 0;
  }

  /* Dark mode support */
  :global([data-theme="dark"]) .dropdown-button,
  :global([data-theme="dark"]) .dropdown-menu {
    background: var(--surface-color, #1f2937);
    border-color: var(--border-color, #374151);
    color: var(--text-color, #f9fafb);
  }

  :global([data-theme="dark"]) .dropdown-button:hover {
    background: var(--surface-hover-color, #374151);
  }

  :global([data-theme="dark"]) .dropdown-header {
    background: var(--surface-secondary-color, #111827);
    border-color: var(--border-color, #374151);
  }

  :global([data-theme="dark"]) .dropdown-item:hover {
    background: var(--surface-hover-color, #374151);
  }

  :global([data-theme="dark"]) .dropdown-item.selected {
    background: rgba(59, 130, 246, 0.2);
  }
</style>
