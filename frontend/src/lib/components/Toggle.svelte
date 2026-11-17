<script lang="ts">
  let {
    checked,
    onchange,
    label,
  }: {
    checked: boolean;
    onchange: (value: boolean) => void;
    label: string;
  } = $props();
</script>

<button
  class="toggle"
  class:on={checked}
  role="switch"
  aria-checked={checked}
  aria-label={label}
  onclick={() => onchange(!checked)}
>
  <i></i>
</button>

<style>
  .toggle {
    position: relative;
    flex: none;
    width: 35px;
    height: 20px;
    padding: 0;
    border: 1px solid var(--border-strong);
    border-radius: 20px;
    background: var(--surface-2);
    transition:
      background 0.18s,
      border-color 0.18s;
  }
  .toggle i {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--text-faint);
    transition:
      left 0.18s,
      background 0.18s;
  }
  .toggle.on {
    border-color: var(--btn-border);
    background: var(--accent);
  }
  .toggle.on i {
    left: 18px;
    background: var(--text-strong);
  }
</style>
