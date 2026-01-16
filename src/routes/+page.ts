import { env } from '$env/dynamic/public';

export async function load({ fetch }) {
	const response = await fetch(`${env.PUBLIC_API_URL}/api/dashboard`);
	const data = await response.json();
	return data;
}
