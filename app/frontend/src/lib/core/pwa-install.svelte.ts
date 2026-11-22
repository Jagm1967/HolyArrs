// Progressive-enhancement web app installer: prefers navigator.install() when
// available and falls back to the single-use beforeinstallprompt event.
interface BeforeInstallPromptEvent extends Event {
	prompt(): Promise<void>;
	readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

type InstallNavigator = Navigator & { install?: () => Promise<void> };

let deferredPrompt = $state<BeforeInstallPromptEvent | undefined>(undefined);
let installed = $state(false);
let busy = $state(false);
let outcome = $state<'ok' | 'cancelled' | 'failed' | ''>('');
let listening = false;

function hasWebInstall(): boolean {
	return typeof Navigator !== 'undefined' && typeof (Navigator.prototype as InstallNavigator).install === 'function';
}

function isStandalone(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia?.('(display-mode: standalone)').matches || (navigator as Navigator & { standalone?: boolean }).standalone === true;
}

export function startInstallCapture(): () => void {
	if (typeof window === 'undefined' || listening) return () => {};
	listening = true;
	const onPrompt = (e: Event) => {
		e.preventDefault();
		deferredPrompt = e as BeforeInstallPromptEvent;
	};
	const onInstalled = () => {
		installed = true;
		deferredPrompt = undefined;
	};
	window.addEventListener('beforeinstallprompt', onPrompt);
	window.addEventListener('appinstalled', onInstalled);
	return () => {
		window.removeEventListener('beforeinstallprompt', onPrompt);
		window.removeEventListener('appinstalled', onInstalled);
		listening = false;
	};
}

export function canInstallApp(): boolean {
	if (installed || isStandalone()) return false;
	return hasWebInstall() || deferredPrompt !== undefined;
}

export function installBusy(): boolean {
	return busy;
}

export function installOutcome(): 'ok' | 'cancelled' | 'failed' | '' {
	return outcome;
}

export async function triggerInstallApp(): Promise<void> {
	if (busy) return;
	busy = true;
	outcome = '';
	try {
		if (hasWebInstall()) {
			const nav = navigator as InstallNavigator;
			await nav.install?.();
			installed = true;
			outcome = 'ok';
		} else if (deferredPrompt) {
			// Discard single-use event before awaiting the user's choice.
			const promptEvent = deferredPrompt;
			deferredPrompt = undefined;
			await promptEvent.prompt();
			const choice = await promptEvent.userChoice;
			if (choice.outcome === 'accepted') {
				installed = true;
				outcome = 'ok';
			} else {
				outcome = 'cancelled';
			}
		}
	} catch (err) {
		if (err instanceof Error && err.name === 'AbortError') {
			outcome = 'cancelled';
		} else {
			outcome = 'failed';
			console.error(err);
		}
	} finally {
		busy = false;
	}
}
