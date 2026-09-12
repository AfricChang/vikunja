import type {APIRequestContext} from '@playwright/test'
import {objectToSnakeCase} from '../../src/helpers/case'

export async function updateUserSettings(apiContext: APIRequestContext, token: string, settings: any) {
	const apiUrl = (process.env.API_URL || 'http://localhost:3456/api/v1').replace(/\/+$/, '')

	const userResponse = await apiContext.get(`${apiUrl}/user`, {
		headers: {
			'Authorization': `Bearer ${token}`,
		},
	})
	if (!userResponse.ok()) {
		throw new Error(`Could not read user settings (${userResponse.status()}): ${await userResponse.text()}`)
	}

	const userData = await userResponse.json()
	// GET /user returns { settings: { frontend_settings: ... }, ... }
	// POST /user/settings/general expects { frontend_settings: ... } at the top level
	const oldSettings = userData.settings || {}

	const snakeSettings = objectToSnakeCase(settings)

	// Deep merge frontend_settings if provided
	const mergedSettings = {
		...oldSettings,
		...snakeSettings,
	}

	if (snakeSettings.frontend_settings) {
		mergedSettings.frontend_settings = {
			...(oldSettings.frontend_settings || {}),
			...snakeSettings.frontend_settings,
		}
	}

	const updateResponse = await apiContext.post(`${apiUrl}/user/settings/general`, {
		headers: {
			'Authorization': `Bearer ${token}`,
		},
		data: mergedSettings,
	})
	if (!updateResponse.ok()) {
		throw new Error(`Could not update user settings (${updateResponse.status()}): ${await updateResponse.text()}`)
	}
}
