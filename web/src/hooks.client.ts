import { checkAuth, redirectToLogin } from '$lib/auth';

export async function init() {
	const user = await checkAuth();

	if (!user) {
		redirectToLogin();
	}
}
