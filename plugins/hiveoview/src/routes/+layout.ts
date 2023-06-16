import { hapi } from '$lib/hapi/HubAPI.js';
import type { ServerLoadEvent } from '@sveltejs/kit';

// import { writable } from 'svelte/store';
// export const isConnected = writable(false);

/** @type {import("./$types").LayoutServerLoad} */
export async function load(ev: ServerLoadEvent) {
	// const loginID = ev.cookies?.get('loginID') || '';
	// const refreshToken = ev.cookies?.get('refreshToken') || '';

	const loginID = 'test';
	const refreshToken = '';

	await hapi.initialize();
	// the hapi server is the same as the ui server address
	await hapi.connect(ev.url.hostname, loginID);

	// isConnected.set(true);

	if (refreshToken) {
		const newRefresh = await hapi.loginRefresh(loginID, refreshToken);
		ev.cookies.set('refreshToken', newRefresh);
	}

	// login with a refresh token
	console.info('constat=', hapi.conStat);
	return { conStat: hapi.conStat };
}
