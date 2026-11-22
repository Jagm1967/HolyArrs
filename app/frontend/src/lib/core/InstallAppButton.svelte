<script lang="ts">
	import { t } from '#lib/core/i18n/i18n.svelte.ts';
	import { canInstallApp, installBusy, installOutcome, triggerInstallApp } from './pwa-install.svelte.ts';

	const visible = $derived(canInstallApp());
	const busy = $derived(installBusy());
	const status = $derived(installOutcome());

	function onClick() {
		if (busy) return;
		void triggerInstallApp();
	}
</script>

{#if visible || status}
	<section class="pwa-card" aria-label={t('settings.account.installTitle')}>
		<div class="pwa-row">
			<div class="pwa-copy">
				<h2 class="pwa-title">{t('settings.account.installTitle')}</h2>
				<p class="pwa-desc">{t('settings.account.installDesc')}</p>
			</div>
			<button
				type="button"
				class="pwa-btn"
				hidden={!visible}
				aria-disabled={busy ? 'true' : undefined}
				onclick={onClick}
			>
				{t('settings.account.installButton')}
			</button>
		</div>
		{#if status}
			<p class="pwa-status" role="status">
				{#if status === 'ok'}
					{t('settings.account.installDone')}
				{:else if status === 'cancelled'}
					{t('settings.account.installCancelled')}
				{:else}
					{t('settings.account.installFailed')}
				{/if}
			</p>
		{/if}
	</section>
{/if}

<style>
	.pwa-card {
		margin-top: 16px;
		padding: 16px 20px;
		border-radius: 12px;
		border: 1px solid var(--border, rgba(128, 128, 128, 0.25));
		background: var(--surface, transparent);
	}
	.pwa-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		flex-wrap: wrap;
	}
	.pwa-title {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
	}
	.pwa-desc {
		margin: 4px 0 0;
		font-size: 0.9rem;
		opacity: 0.78;
	}
	.pwa-btn {
		padding: 8px 16px;
		border: 0;
		border-radius: 8px;
		background: #d42d24;
		color: #fff;
		font-weight: 600;
		cursor: pointer;
	}
	.pwa-btn[aria-disabled='true'] {
		opacity: 0.6;
		cursor: progress;
	}
	.pwa-status {
		margin: 10px 0 0;
		font-size: 0.875rem;
		opacity: 0.85;
	}
</style>
