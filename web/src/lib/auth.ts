// Auth utilities for session validation and login redirects

export async function checkAuth(): Promise<{ user_id: string } | null> {
	try {
		const resp = await fetch('/api/me', {
			credentials: 'include'
		});

		if (!resp.ok) {
			return null;
		}

		return await resp.json();
	} catch (err) {
		console.error('Auth check failed:', err);
		return null;
	}
}

export function redirectToLogin() {
	const authURL = (window as any).__AUTH_URL__ || '/login';
	const redirectParam = encodeURIComponent(window.location.href);
	window.location.href = `${authURL}?redirect=${redirectParam}`;
}

export async function logout(): Promise<void> {
	await fetch('/api/logout', { method: 'POST', credentials: 'include' });
	redirectToLogin();
}

export async function authenticatedFetch(url: string, options: RequestInit = {}): Promise<Response> {
	const resp = await fetch(url, {
		...options,
		credentials: 'include'
	});

	if (resp.status === 401) {
		redirectToLogin();
		throw new Error('Unauthorized');
	}

	return resp;
}
