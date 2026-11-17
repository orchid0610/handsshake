<script lang="ts">
  import Icon, { type IconName } from "./Icon.svelte";

  let {
    icon,
    eyebrow,
    title,
    text,
    tag = "",
    action = "",
    disabled = false,
    onaction,
  }: {
    icon: IconName;
    eyebrow: string;
    title: string;
    text: string;
    tag?: string;
    action?: string;
    disabled?: boolean;
    onaction?: () => void;
  } = $props();
</script>

<div class="empty">
  <span class="icon"><Icon name={icon} size={23} /></span>
  <p class="eyebrow">{eyebrow}</p>
  <h1 class="page-title">
    {title}
    {#if tag}<span class="tag">{tag}</span>{/if}
  </h1>
  <p class="text">{text}</p>
  {#if action}
    <button
      class="btn btn-primary"
      {disabled}
      onclick={onaction}
      title={disabled ? "Not available in this release" : undefined}
    >
      {action}
    </button>
  {/if}
</div>

<style>
  .empty {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px;
    text-align: center;
  }
  .icon {
    display: grid;
    place-items: center;
    width: 50px;
    height: 50px;
    margin-bottom: 18px;
    color: var(--accent-text);
    border: 1px solid var(--accent-border);
    border-radius: 14px;
    background: var(--accent-soft);
  }
  h1 {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 6px;
  }
  .tag {
    padding: 3px 9px;
    border-radius: 99px;
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--accent-text);
    background: var(--accent-soft);
    border: 1px solid var(--accent-border);
  }
  .text {
    max-width: 360px;
    margin: 10px 0 22px;
    color: var(--text-dim);
    font-size: 13px;
    line-height: 1.6;
  }
</style>
